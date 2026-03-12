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
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"
	"golang.org/x/sync/errgroup"
)

type PreviewFile struct {
	FilePath           string               `json:"file_path,omitempty" xml:"file_path"`
	SingleFilePreviews []*SingleFilePreview `json:"single_file_previews,omitempty" xml:"single_file_previews>single_file_preview"`
}

type SingleFilePreview struct {
	SheetName   string         `json:"sheet_name,omitempty" xml:"sheet_name"`
	Header      []*ExcelCell   `json:"header" xml:"header"`
	Content     [][]*ExcelCell `json:"content,omitempty" xml:"content"`
	MergedCells []*ExcelCell   `json:"merged_cells,omitempty" xml:"merged_cells>merged_cell"`
}

type ExcelCell struct {
	Address string `json:"address,omitempty" xml:"address"` // B1:D1 表示 B1 到 D1 范围的单元格, B1 表示单个单元格
	Value   string `json:"value,omitempty" xml:"value"`     // 单元格的值
}

func PreviewPath(path string) ([]*PreviewFile, error) {
	filePaths, err := getAllFiles(path)
	if err != nil {
		return nil, err
	}
	if len(filePaths) == 0 {
		return nil, nil
	}
	resp := make([]*PreviewFile, len(filePaths))
	eg := errgroup.Group{}
	eg.SetLimit(10)
	for i := range filePaths {
		idx := i
		fp := filePaths[idx]
		eg.Go(func() error {
			var (
				pf *PreviewFile
				e  error
			)
			if ext := filepath.Ext(fp); ext != ".xlsx" { // .xls not support
				pf = &PreviewFile{FilePath: fp}
			} else {
				pf, e = previewExcelDocument(path)
			}
			if e != nil {
				return e
			}
			resp[idx] = pf
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return resp, nil
}

func getAllFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		if isHiddenFile(info.Name()) {
			return nil, nil
		}
		return []string{path}, nil
	}

	var files []string
	if err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if isHiddenFile(info.Name()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return files, nil
}

func previewExcelDocument(filePath string) (*PreviewFile, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	pf := &PreviewFile{
		FilePath:           filePath,
		SingleFilePreviews: nil,
	}
	for _, sheetName := range f.GetSheetList() {
		sfp, err := parseSheet(f, sheetName)
		if err != nil {
			return nil, err
		}

		pf.SingleFilePreviews = append(pf.SingleFilePreviews, sfp)
	}

	return pf, nil
}

func parseSheet(f *excelize.File, sheetName string) (*SingleFilePreview, error) {
	preview := &SingleFilePreview{
		SheetName:   sheetName,
		Header:      make([]*ExcelCell, 0),
		Content:     make([][]*ExcelCell, 0),
		MergedCells: make([]*ExcelCell, 0),
	}
	mcs := make([][][]int, 20)
	mergedCells, err := f.GetMergeCells(sheetName)
	if err != nil {
		return nil, err
	}
	for _, cell := range mergedCells {
		lcol, lrow, err := excelize.CellNameToCoordinates(cell.GetStartAxis())
		if err != nil {
			return nil, err
		}
		if lrow >= 20 {
			continue
		}

		rcol, rrow, err := excelize.CellNameToCoordinates(cell.GetEndAxis())
		if err != nil {
			return nil, err
		}
		preview.MergedCells = append(preview.MergedCells, &ExcelCell{
			Address: cell[0],
			Value:   cell.GetCellValue(),
		})
		r := mcs[lrow]
		r = append(r, []int{lcol, lrow, rcol, rrow})
	}

	rowIter, err := f.Rows(sheetName)
	if err != nil {
		return nil, err
	}

	i := 0
	for rowIter.Next() && i < 20 {
		values, err := rowIter.Columns()
		if err != nil {
			return nil, err
		}
		var rowValues []*ExcelCell
		for j, val := range values {
			col, row := j+1, i+1
			skip := false
			if r := mcs[i]; len(r) > 0 {
				for _, coverage := range r {
					if col >= coverage[0] && row >= coverage[1] && col <= coverage[2] && row <= coverage[3] {
						skip = true
						break
					}
				}
			}
			if skip {
				continue
			}
			addr, err := excelize.CoordinatesToCellName(col, row)
			if err != nil {
				return nil, err
			}
			rowValues = append(rowValues, &ExcelCell{Address: addr, Value: val})
		}
		if i == 0 {
			preview.Header = rowValues
		} else {
			preview.Content = append(preview.Content, rowValues)
		}
		i++
	}

	return preview, nil
}

func isHiddenFile(name string) bool {
	return len(name) > 0 && name[0] == '.'
}
