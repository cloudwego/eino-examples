/*
 * Copyright 2025 CloudWeGo Authors
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
 */

package generic

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestPreviewPathDirectoryWithExcelFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "sample.xlsx")

	f := excelize.NewFile()
	t.Cleanup(func() {
		_ = f.Close()
	})
	if err := f.SetCellValue("Sheet1", "A1", "name"); err != nil {
		t.Fatal(err)
	}
	if err := f.SetCellValue("Sheet1", "A2", "Alice"); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs(filePath); err != nil {
		t.Fatal(err)
	}

	previews, err := PreviewPath(dir)
	if err != nil {
		t.Fatalf("PreviewPath() error = %v", err)
	}
	if len(previews) != 1 {
		t.Fatalf("len(previews) = %d, want 1", len(previews))
	}

	preview := previews[0]
	if preview.FilePath != filePath {
		t.Fatalf("preview.FilePath = %q, want %q", preview.FilePath, filePath)
	}
	if len(preview.SingleFilePreviews) != 1 {
		t.Fatalf("len(preview.SingleFilePreviews) = %d, want 1", len(preview.SingleFilePreviews))
	}
	if got := preview.SingleFilePreviews[0].Header[0].Value; got != "name" {
		t.Fatalf("header value = %q, want %q", got, "name")
	}
}
