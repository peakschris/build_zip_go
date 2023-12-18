// Copyright 2021 The Bazel Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Original: https://github.com/bazelbuild/rules_pkg/blob/main/pkg/private/zip/build_zip.py
// Converted to Go by Github Copilot
package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
)

// These must be kept in sync with the declarations in private/pkg_files.bzl
const (
    ENTRY_IS_FILE       = "file"        // Entry is a file: take content from <src>
    ENTRY_IS_LINK       = "symlink"     // Entry is a symlink: dest -> <src>
    ENTRY_IS_DIR        = "dir"         // Entry is an empty dir
    ENTRY_IS_TREE       = "tree"        // Entry is a tree artifact: take tree from <src>
    ENTRY_IS_EMPTY_FILE = "empty-file"  // Entry is a an empty file
)

type ManifestEntry struct {
    Type   string `json:"type"`
    Dest   string `json:"dest"`
    Src    string `json:"src"`
    Mode   string `json:"mode"`
    User   string `json:"user"`
    Group  string `json:"group"`
    UID    int    `json:"uid"`
    GID    int    `json:"gid"`
    Origin string `json:"origin,omitempty"`
}

func readEntriesFrom(data []byte) ([]ManifestEntry, error) {
    var entries []ManifestEntry
    err := json.Unmarshal(data, &entries)
    if err != nil {
        return nil, err
    }
    return entries, nil
}

func ReadEntriesFromFile(manifestPath string) ([]ManifestEntry, error) {
    data, err := ioutil.ReadFile(manifestPath)
    if err != nil {
        return nil, err
    }
    return readEntriesFrom(data)
}

func entryTypeToString(et string) (string, error) {
    switch et {
    case ENTRY_IS_FILE:
        return "file", nil
    case ENTRY_IS_LINK:
        return "symlink", nil
    case ENTRY_IS_DIR:
        return "directory", nil
    case ENTRY_IS_TREE:
        return "tree", nil
    case ENTRY_IS_EMPTY_FILE:
        return "empty_file", nil
    default:
        return "", fmt.Errorf("Invalid entry id %s", et)
    }
}
