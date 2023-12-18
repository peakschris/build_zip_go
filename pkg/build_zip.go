// Copyright 2015 The Bazel Authors. All rights reserved.
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
    "archive/zip"
    "flag"
    "fmt"
    "io"
    "log"
    "os"
    "path/filepath"
    "sort"
    "strconv"
    "strings"
    "time"
)

const (
    zipEpoch      = 315532800
    unixDirBit    = 040000
    msdosDirBit   = 0x10
    unixSymlinkBit = 0120000
)

func main() {
    output := flag.String("o", "", "The output zip file path.")
    directory := flag.String("d", "/", "An absolute path to use as a prefix for all files in the zip.")
    timestamp := flag.Int("t", zipEpoch, "The unix time to use for files added into the zip. values prior to Jan 1, 1980 are ignored.")
    stampFrom := flag.String("stamp_from", "", "File to find BUILD_STAMP in")
    mode := flag.String("m", "", "The file system mode to use for files added into the zip.")
    compressionType := flag.String("c", "", "The compression type to use")
    compressionLevel := flag.String("l", "", "The compression level to use")
    manifest := flag.String("manifest", "", "manifest of contents to add to the layer.")
    flag.Parse()

    unixTs := max(zipEpoch, *timestamp)
    if *stampFrom != "" {
        unixTs, _ = GetTimestamp(*stampFrom)
    }

    manifestEntries := loadManifest(*directory, *manifest)
    zipWriter, err := NewZipWriter(*output, unixTs, *mode, *compressionType, *compressionLevel)
    if err != nil {
        log.Fatal(err)
    }
    defer zipWriter.Close()

    for _, entry := range manifestEntries {
        err = zipWriter.AddManifestEntry(entry)
        if err != nil {
          log.Fatal(err)
        }
    }
}

func combinePaths(left, right string) string {
  result := strings.TrimRight(left, "/") + "/" + strings.TrimLeft(right, "/")
  return strings.TrimLeft(result, "/")
}

// loadManifest loads the manifest entries from a file and combines them with the provided prefix.
func loadManifest(prefix string, manifestPath string) []*ManifestEntry {
  manifestMap := make(map[string]*ManifestEntry)

  entries, _ := ReadEntriesFromFile(manifestPath)
  for _, entry := range entries {
    entry.Dest = combinePaths(prefix, entry.Dest)
    manifestMap[entry.Dest] = &ManifestEntry{
      Type:   entry.Type,
      Dest:   entry.Dest,
      Src:    entry.Src,
      Mode:   entry.Mode,
      User:   entry.User,
      Group:  entry.Group,
      UID:    entry.UID,
      GID:    entry.GID,
      Origin: entry.Origin,
    }
  }

  manifestKeys := make([]string, 0, len(manifestMap))
  for dest := range manifestMap {
    manifestKeys = append(manifestKeys, dest)
  }

  for _, dest := range manifestKeys {
    parent := dest
    for parent != "." {
      parent = filepath.Dir(parent)
      if parent != "." && manifestMap[parent] == nil {
        manifestMap[parent] = &ManifestEntry{
          Type:   ENTRY_IS_DIR,
          Dest:   parent,
          Src:    "",
          Mode:   "0o755",
          User:   "",
          Group:  "",
          UID:    0,
          GID:    0,
          Origin: fmt.Sprintf("parent directory of %s", manifestMap[dest].Origin),
        }
      }
    }
  }

  manifestEntries := make([]*ManifestEntry, 0, len(manifestMap))
  for _, entry := range manifestMap {
    fmt.Printf("%s\n", entry.Dest)
    manifestEntries = append(manifestEntries, entry)
  }

  sort.Slice(manifestEntries, func(i, j int) bool {
    return manifestEntries[i].Dest < manifestEntries[j].Dest
  })

  return manifestEntries
}



type ZipWriter struct {
    outputPath      string
    timeStamp       int
    defaultMode     string
    compressionType uint16
    compressionLevel uint16
    zipWriter       *zip.Writer
}

func NewZipWriter(outputPath string, timeStamp int, defaultMode string, compressionType string, compressionLevel string) (*ZipWriter, error) {
    zipWriter := &ZipWriter{
        outputPath:  outputPath,
        timeStamp:   timeStamp,
        defaultMode: defaultMode,
    }

    compression, err := parseCompression(compressionType)
    if err != nil {
        return nil, err
    }
    zipWriter.compressionType = compression

    level, err := parseCompressionLevel(compressionLevel)
    if err != nil {
        return nil, err
    }
    zipWriter.compressionLevel = level

    fmt.Printf("%s\n", outputPath)
    file, err := os.Create(outputPath)
    if err != nil {
        return nil, err
    }
    fmt.Printf("Created %s\n", outputPath)
    zipWriter.zipWriter = zip.NewWriter(file)

    return zipWriter, nil
}

func (zw *ZipWriter) Close() error {
    return zw.zipWriter.Close()
}

func (zw *ZipWriter) makeZipInfo(path string, mode string) *zip.FileHeader {
  entryInfo := &zip.FileHeader{
    Name:     path,
    Modified: time.Unix(int64(zw.timeStamp), 0),
    Flags:    0x800,
  }

  if mode != "" {
    fMode, _ := strconv.ParseUint(mode, 8, 32)
    entryInfo.ExternalAttrs = uint32(fMode) << 16
  } else {
    entryInfo.ExternalAttrs = uint32(parseMode(zw.defaultMode)) << 16
  }

  return entryInfo
}

// Add an entry to the zip file.
func (zw *ZipWriter) AddManifestEntry(entry *ManifestEntry) error {
  entryType := entry.Type
  dest := entry.Dest
  src := entry.Src
  mode := entry.Mode
  //user := entry.User
  //group := entry.Group

  // Use the pkg_tar mode/owner remapping as a fallback
  dstPath := strings.TrimSuffix(dest, "/")
  if entryType == ENTRY_IS_DIR && !strings.HasSuffix(dstPath, "/") {
    dstPath += "/"
  }
  entryInfo := zw.makeZipInfo(dstPath, mode)

  if entryType == ENTRY_IS_FILE {
    entryInfo.Method = zw.compressionType
    // Using utf-8 for the file names is for Go <1.16 compatibility.
    srcContent, err := os.Open(src)
    if err != nil {
      return err
    }
    defer srcContent.Close()

    writer, err := zw.zipWriter.CreateHeader(entryInfo)
    if err != nil {
      return err
    }

    _, err = io.Copy(writer, srcContent)
    if err != nil {
      return err
    }
  } else if entryType == ENTRY_IS_TREE {
    return zw.AddTree(src, dest, mode)
  }

  return nil
}

func (zw *ZipWriter) AddTree(treeTop string, destPath string, mode string) error {
  treeTop = filepath.Clean(treeTop)
  dest := strings.TrimSuffix(destPath, "/")
  dest = filepath.Clean(dest)

  toWrite := make(map[string]string)
  err := filepath.Walk(treeTop, func(path string, info os.FileInfo, err error) error {
      if err != nil {
          return err
      }

      relPathFromTop, err := filepath.Rel(treeTop, path)
      if err != nil {
          return err
      }

      destDir := filepath.Join(dest, relPathFromTop)
      toWrite[destDir] = ""

      if !info.IsDir() {
          toWrite[filepath.Join(destDir, info.Name())] = path
      }

      return nil
  })
  if err != nil {
      return err
  }

  for path, contentPath := range toWrite {
      if contentPath != "" {
          fMode := mode
          if mode == "" {
              if _, err := os.Stat(contentPath); err == nil {
                  fMode = "0755"
              } else {
                  fMode = zw.defaultMode
              }
          }

          info := zw.makeZipInfo(path, fMode)

          srcContent, err := os.Open(contentPath)
          if err != nil {
            return err
          }
          defer srcContent.Close()

          writer, err := zw.zipWriter.CreateHeader(info)
          if err != nil {
            return err
          }

          _, err = io.Copy(writer, srcContent)
          if err != nil {
            return err
          }
      } else {
          // Implicitly created directory
          dirPath := path
          if !strings.HasSuffix(dirPath, "/") {
              dirPath += "/"
          }

          info := zw.makeZipInfo(dirPath, "0755")
          info.Method = zip.Store

          // ignoring writer return
          _, err := zw.zipWriter.CreateHeader(info)
          if err != nil {
            return err
          }
      }
  }

  return nil
}

func parseMode(mode string) uint32 {
    if mode == "" {
        return 0
    }

    parsedMode, err := strconv.ParseInt(mode, 8, 32)
    if err != nil {
        log.Fatalf("invalid mode: %s", mode)
    }

    return uint32(parsedMode)
}

func parseCompression(compressionType string) (uint16, error) {
    switch compressionType {
    case "deflated":
        return zip.Deflate, nil
    case "stored":
        return zip.Store, nil
    case "":
      return zip.Deflate, nil
    default:
        return 0, fmt.Errorf("invalid compression type: %s", compressionType)
    }
}

func parseCompressionLevel(compressionLevel string) (uint16, error) {
    if compressionLevel == "" {
        return zip.Store, nil
    }

    parsedLevel, err := strconv.Atoi(compressionLevel)
    if err != nil {
        return 0, fmt.Errorf("invalid compression level: %s", compressionLevel)
    }

    return uint16(parsedLevel), nil
}
