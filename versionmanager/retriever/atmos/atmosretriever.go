/*
 *
 * Copyright 2024 tofuutils authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package atmosretriever

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hashicorp/go-hclog"

	"github.com/tofuutils/tenv/v4/config"
	"github.com/tofuutils/tenv/v4/config/cmdconst"
	"github.com/tofuutils/tenv/v4/config/envname"
	"github.com/tofuutils/tenv/v4/pkg/apimsg"
	sha256check "github.com/tofuutils/tenv/v4/pkg/check/sha256"
	"github.com/tofuutils/tenv/v4/pkg/download"
	"github.com/tofuutils/tenv/v4/pkg/github"
	"github.com/tofuutils/tenv/v4/pkg/winbin"
	htmlretriever "github.com/tofuutils/tenv/v4/versionmanager/retriever/html"
)

const (
	baseFileName   = "atmos_"
	cloudposseName = "cloudposse"

	rwePerm = 0o755
)

type AtmosRetriever struct {
	conf *config.Config
}

func Make(conf *config.Config) AtmosRetriever {
	return AtmosRetriever{conf: conf}
}

func (r AtmosRetriever) Install(ctx context.Context, versionStr string, targetPath string) error {
	err := r.conf.InitRemoteConf()
	if err != nil {
		return err
	}

	tag := versionStr
	// assume that atmos tags start with a 'v'
	// and version in asset name does not
	if tag[0] == 'v' {
		versionStr = versionStr[1:]
	} else {
		tag = "v" + versionStr
	}

	var assetURLs []string
	fileName, shaFileName := buildAssetNames(versionStr, r.conf.Arch)
	if r.conf.Displayer.IsDebug() {
		r.conf.Displayer.Log(hclog.Debug, apimsg.MsgSearch, apimsg.AssetsName, []string{fileName, shaFileName})
	}

	switch r.conf.Atmos.GetInstallMode() {
	case config.InstallModeDirect:
		baseAssetURL, err2 := url.JoinPath(r.conf.Atmos.GetRemoteURL(), cloudposseName, cmdconst.AtmosName, github.Releases, github.Download, tag)
		if err2 != nil {
			return err2
		}

		assetURLs, err = htmlretriever.BuildAssetURLs(baseAssetURL, fileName, shaFileName)
	case config.ModeAPI:
		assetURLs, err = github.AssetDownloadURL(ctx, tag, []string{fileName, shaFileName}, r.conf.Atmos.GetRemoteURL(), r.conf.GithubToken, r.conf.Displayer.Display)
	default:
		return config.ErrInstallMode
	}
	if err != nil {
		return err
	}

	assetURLs, err = download.ApplyURLTransformer(r.conf.Atmos.GetRewriteRule(), assetURLs...)
	if err != nil {
		return err
	}

	requestOptions := config.GetBasicAuthOption(r.conf.Getenv, envname.AtmosRemoteUser, envname.AtmosRemotePass)
	data, err := download.Bytes(ctx, assetURLs[0], r.conf.Displayer.Display, download.NoCheck, requestOptions...)
	if err != nil {
		return err
	}

	dataSums, err := download.Bytes(ctx, assetURLs[1], r.conf.Displayer.Display, download.NoCheck, requestOptions...)
	if err != nil {
		return err
	}

	if err = sha256check.Check(data, dataSums, fileName); err != nil {
		return err
	}

	err = os.MkdirAll(targetPath, rwePerm)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(targetPath, winbin.GetBinaryName(cmdconst.AtmosName)), data, rwePerm)
}

func (r AtmosRetriever) ListVersions(ctx context.Context) ([]string, error) {
	err := r.conf.InitRemoteConf()
	if err != nil {
		return nil, err
	}

	requestOptions := config.GetBasicAuthOption(r.conf.Getenv, envname.AtmosRemoteUser, envname.AtmosRemotePass)

	listURL := r.conf.Atmos.GetListURL()
	switch r.conf.Atmos.GetListMode() {
	case config.ListModeHTML:
		baseURL, err := url.JoinPath(listURL, cloudposseName, cmdconst.AtmosName, github.Releases, github.Download)
		if err != nil {
			return nil, err
		}

		r.conf.Displayer.Display(apimsg.MsgFetchAllReleases + baseURL)

		return htmlretriever.ListReleases(ctx, baseURL, r.conf.Atmos.Data, requestOptions)
	case config.ModeAPI:
		r.conf.Displayer.Display(apimsg.MsgFetchAllReleases + listURL)

		return github.ListReleases(ctx, listURL, r.conf.GithubToken)
	default:
		return nil, config.ErrListMode
	}
}

func buildAssetNames(version string, arch string) (string, string) {
	var nameBuilder strings.Builder
	nameBuilder.WriteString(baseFileName)
	nameBuilder.WriteString(version)
	nameBuilder.WriteByte('_')
	sumsAssetName := nameBuilder.String() + "SHA256SUMS"

	nameBuilder.WriteString(runtime.GOOS)
	nameBuilder.WriteByte('_')
	nameBuilder.WriteString(arch)
	_, _ = winbin.WriteSuffixTo(&nameBuilder)

	return nameBuilder.String(), sumsAssetName
}
