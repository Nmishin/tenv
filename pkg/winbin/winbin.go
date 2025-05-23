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

package winbin

import (
	"io"
	"runtime"
)

const (
	suffix      = ".exe"
	osName      = "windows"
	zipSuffix   = ".zip"
	tarGzSuffix = ".tar.gz"
)

func GetArchiveFormat() string {
	if runtime.GOOS == osName {
		return zipSuffix
	}

	return tarGzSuffix
}

func GetBinaryName(execName string) string {
	if runtime.GOOS != osName {
		return execName
	}

	return execName + suffix
}

func WriteSuffixTo(writer io.StringWriter) (int, error) {
	if runtime.GOOS != osName {
		return 0, nil
	}

	return writer.WriteString(suffix)
}
