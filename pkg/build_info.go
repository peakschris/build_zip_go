// Copyright 2021 The Bazel Authors. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//    http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
    "fmt"
    "io/ioutil"
    "strconv"
    "strings"
)

func GetTimestamp(volatileStatusFile string) (int, error) {
    content, err := ioutil.ReadFile(volatileStatusFile)
    if err != nil {
        return 0, err
    }

    lines := strings.Split(string(content), "\n")
    for _, line := range lines {
        parts := strings.Split(strings.TrimSpace(line), " ")
        if len(parts) > 1 && parts[0] == "BUILD_TIMESTAMP" {
            timestamp, err := strconv.Atoi(parts[1])
            if err != nil {
                return 0, err
            }
            return timestamp, nil
        }
    }

    return 0, fmt.Errorf("Invalid status file <%s>. Expected to find BUILD_TIMESTAMP", volatileStatusFile)
}
