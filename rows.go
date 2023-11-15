// Copyright 2016 - 2023 The excelize Authors. All rights reserved. Use of
// this source code is governed by a BSD-style license that can be found in
// the LICENSE file.
//
// package excelize_ch providing a set of functions that allow you to write to and
// read from XLAM / XLSM / XLSX / XLTM / XLTX files. Supports reading and
// writing spreadsheet documents generated by Microsoft Excel™ 2007 and later.
// Supports complex components by high compatibility, and provided streaming
// API for generating or reading data from a worksheet with huge amounts of
// data. This library needs Go version 1.16 or later.

package excelize_ch

import (
	"bytes"
	"encoding/xml"
	"io"
	"math"
	"os"
	"strconv"

	"github.com/mohae/deepcopy"
)

// GetRows return all the rows in a sheet by given worksheet name, returned as
// a two-dimensional array, where the value of the cell is converted to the
// string type. If the cell format can be applied to the value of the cell,
// the applied value will be used, otherwise the original value will be used.
// GetRows fetched the rows with value or formula cells, the continually blank
// cells in the tail of each row will be skipped, so the length of each row
// may be inconsistent.
//
// For example, get and traverse the value of all cells by rows on a worksheet
// named 'Sheet1':
//
//	rows, err := f.GetRows("Sheet1")
//	if err != nil {
//	    fmt.Println(err)
//	    return
//	}
//	for _, row := range rows {
//	    for _, colCell := range row {
//	        fmt.Print(colCell, "\t")
//	    }
//	    fmt.Println()
//	}
func (f *File) GetRows(sheet string, opts ...Options) ([][]string, error) {
	rows, err := f.Rows(sheet)
	if err != nil {
		return nil, err
	}
	results, cur, max := make([][]string, 0, 64), 0, 0
	for rows.Next() {
		cur++
		row, err := rows.Columns(opts...)
		if err != nil {
			break
		}
		results = append(results, row)
		if len(row) > 0 {
			max = cur
		}
	}
	return results[:max], rows.Close()
}

// Rows defines an iterator to a sheet.
type Rows struct {
	err                     error
	curRow, seekRow         int
	needClose, rawCellValue bool
	sheet                   string
	f                       *File
	tempFile                *os.File
	sst                     *xlsxSST
	decoder                 *xml.Decoder
	token                   xml.Token
	curRowOpts, seekRowOpts RowOpts
}

// Next will return true if it finds the next row element.
func (rows *Rows) Next() bool {
	rows.seekRow++
	if rows.curRow >= rows.seekRow {
		rows.curRowOpts = rows.seekRowOpts
		return true
	}
	for {
		token, _ := rows.decoder.Token()
		if token == nil {
			return false
		}
		switch xmlElement := token.(type) {
		case xml.StartElement:
			if xmlElement.Name.Local == "row" {
				rows.curRow++
				if rowNum, _ := attrValToInt("r", xmlElement.Attr); rowNum != 0 {
					rows.curRow = rowNum
				}
				rows.token = token
				rows.curRowOpts = extractRowOpts(xmlElement.Attr)
				return true
			}
		case xml.EndElement:
			if xmlElement.Name.Local == "sheetData" {
				return false
			}
		}
	}
}

// GetRowOpts will return the RowOpts of the current row.
func (rows *Rows) GetRowOpts() RowOpts {
	return rows.curRowOpts
}

// Error will return the error when the error occurs.
func (rows *Rows) Error() error {
	return rows.err
}

// Close closes the open worksheet XML file in the system temporary
// directory.
func (rows *Rows) Close() error {
	if rows.tempFile != nil {
		return rows.tempFile.Close()
	}
	return nil
}

// Columns return the current row's column values. This fetches the worksheet
// data as a stream, returns each cell in a row as is, and will not skip empty
// rows in the tail of the worksheet.
func (rows *Rows) Columns(opts ...Options) ([]string, error) {
	if rows.curRow > rows.seekRow {
		return nil, nil
	}
	var rowIterator rowXMLIterator
	var token xml.Token
	rows.rawCellValue = getOptions(opts...).RawCellValue
	if rows.sst, rowIterator.err = rows.f.sharedStringsReader(); rowIterator.err != nil {
		return rowIterator.cells, rowIterator.err
	}
	for {
		if rows.token != nil {
			token = rows.token
		} else if token, _ = rows.decoder.Token(); token == nil {
			break
		}
		switch xmlElement := token.(type) {
		case xml.StartElement:
			rowIterator.inElement = xmlElement.Name.Local
			if rowIterator.inElement == "row" {
				rowNum := 0
				if rowNum, rowIterator.err = attrValToInt("r", xmlElement.Attr); rowNum != 0 {
					rows.curRow = rowNum
				} else if rows.token == nil {
					rows.curRow++
				}
				rows.token = token
				rows.seekRowOpts = extractRowOpts(xmlElement.Attr)
				if rows.curRow > rows.seekRow {
					rows.token = nil
					return rowIterator.cells, rowIterator.err
				}
			}
			if rows.rowXMLHandler(&rowIterator, &xmlElement, rows.rawCellValue); rowIterator.err != nil {
				rows.token = nil
				return rowIterator.cells, rowIterator.err
			}
			rows.token = nil
		case xml.EndElement:
			if xmlElement.Name.Local == "sheetData" {
				return rowIterator.cells, rowIterator.err
			}
		}
	}
	return rowIterator.cells, rowIterator.err
}

// extractRowOpts extract row element attributes.
func extractRowOpts(attrs []xml.Attr) RowOpts {
	rowOpts := RowOpts{Height: defaultRowHeight}
	if styleID, err := attrValToInt("s", attrs); err == nil && styleID > 0 && styleID < MaxCellStyles {
		rowOpts.StyleID = styleID
	}
	if hidden, err := attrValToBool("hidden", attrs); err == nil {
		rowOpts.Hidden = hidden
	}
	if height, err := attrValToFloat("ht", attrs); err == nil {
		rowOpts.Height = height
	}
	return rowOpts
}

// appendSpace append blank characters to slice by given length and source slice.
func appendSpace(l int, s []string) []string {
	for i := 1; i < l; i++ {
		s = append(s, "")
	}
	return s
}

// rowXMLIterator defined runtime use field for the worksheet row SAX parser.
type rowXMLIterator struct {
	err              error
	inElement        string
	cellCol, cellRow int
	cells            []string
}

// rowXMLHandler parse the row XML element of the worksheet.
func (rows *Rows) rowXMLHandler(rowIterator *rowXMLIterator, xmlElement *xml.StartElement, raw bool) {
	if rowIterator.inElement == "c" {
		rowIterator.cellCol++
		colCell := xlsxC{}
		_ = rows.decoder.DecodeElement(&colCell, xmlElement)
		if colCell.R != "" {
			if rowIterator.cellCol, _, rowIterator.err = CellNameToCoordinates(colCell.R); rowIterator.err != nil {
				return
			}
		}
		blank := rowIterator.cellCol - len(rowIterator.cells)
		if val, _ := colCell.getValueFrom(rows.f, rows.sst, raw); val != "" || colCell.F != nil {
			rowIterator.cells = append(appendSpace(blank, rowIterator.cells), val)
		}
	}
}

// Rows returns a rows iterator, used for streaming reading data for a
// worksheet with a large data. This function is concurrency safe. For
// example:
//
//	rows, err := f.Rows("Sheet1")
//	if err != nil {
//	    fmt.Println(err)
//	    return
//	}
//	for rows.Next() {
//	    row, err := rows.Columns()
//	    if err != nil {
//	        fmt.Println(err)
//	    }
//	    for _, colCell := range row {
//	        fmt.Print(colCell, "\t")
//	    }
//	    fmt.Println()
//	}
//	if err = rows.Close(); err != nil {
//	    fmt.Println(err)
//	}
func (f *File) Rows(sheet string) (*Rows, error) {
	if err := checkSheetName(sheet); err != nil {
		return nil, err
	}
	name, ok := f.getSheetXMLPath(sheet)
	if !ok {
		return nil, ErrSheetNotExist{sheet}
	}
	if worksheet, ok := f.Sheet.Load(name); ok && worksheet != nil {
		ws := worksheet.(*xlsxWorksheet)
		ws.mu.Lock()
		defer ws.mu.Unlock()
		// Flush data
		output, _ := xml.Marshal(ws)
		f.saveFileList(name, f.replaceNameSpaceBytes(name, output))
	}
	var err error
	rows := Rows{f: f, sheet: name}
	rows.needClose, rows.decoder, rows.tempFile, err = f.xmlDecoder(name)
	return &rows, err
}

// getFromStringItem build shared string item offset list from system temporary
// file at one time, and return value by given to string index.
func (f *File) getFromStringItem(index int) string {
	if f.sharedStringTemp != nil {
		if len(f.sharedStringItem) <= index {
			return strconv.Itoa(index)
		}
		offsetRange := f.sharedStringItem[index]
		buf := make([]byte, offsetRange[1]-offsetRange[0])
		if _, err := f.sharedStringTemp.ReadAt(buf, int64(offsetRange[0])); err != nil {
			return strconv.Itoa(index)
		}
		return string(buf)
	}
	needClose, decoder, tempFile, err := f.xmlDecoder(defaultXMLPathSharedStrings)
	if needClose && err == nil {
		defer func() {
			err = tempFile.Close()
		}()
	}
	f.sharedStringItem = [][]uint{}
	f.sharedStringTemp, _ = os.CreateTemp(os.TempDir(), "excelize-")
	f.tempFiles.Store(defaultTempFileSST, f.sharedStringTemp.Name())
	var (
		inElement string
		i, offset uint
	)
	for {
		token, _ := decoder.Token()
		if token == nil {
			break
		}
		switch xmlElement := token.(type) {
		case xml.StartElement:
			inElement = xmlElement.Name.Local
			if inElement == "si" {
				si := xlsxSI{}
				_ = decoder.DecodeElement(&si, &xmlElement)

				startIdx := offset
				n, _ := f.sharedStringTemp.WriteString(si.String())
				offset += uint(n)
				f.sharedStringItem = append(f.sharedStringItem, []uint{startIdx, offset})
				i++
			}
		}
	}
	return f.getFromStringItem(index)
}

// xmlDecoder creates XML decoder by given path in the zip from memory data
// or system temporary file.
func (f *File) xmlDecoder(name string) (bool, *xml.Decoder, *os.File, error) {
	var (
		content  []byte
		err      error
		tempFile *os.File
	)
	if content = f.readXML(name); len(content) > 0 {
		return false, f.xmlNewDecoder(bytes.NewReader(content)), tempFile, err
	}
	tempFile, err = f.readTemp(name)
	return true, f.xmlNewDecoder(tempFile), tempFile, err
}

// SetRowHeight provides a function to set the height of a single row. For
// example, set the height of the first row in Sheet1:
//
//	err := f.SetRowHeight("Sheet1", 1, 50)
func (f *File) SetRowHeight(sheet string, row int, height float64) error {
	if row < 1 {
		return newInvalidRowNumberError(row)
	}
	if height > MaxRowHeight {
		return ErrMaxRowHeight
	}
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}

	ws.prepareSheetXML(0, row)

	rowIdx := row - 1
	ws.SheetData.Row[rowIdx].Ht = float64Ptr(height)
	ws.SheetData.Row[rowIdx].CustomHeight = true
	return nil
}

// getRowHeight provides a function to get row height in pixels by given sheet
// name and row number.
func (f *File) getRowHeight(sheet string, row int) int {
	ws, _ := f.workSheetReader(sheet)
	ws.mu.Lock()
	defer ws.mu.Unlock()
	for i := range ws.SheetData.Row {
		v := &ws.SheetData.Row[i]
		if v.R == row && v.Ht != nil {
			return int(convertRowHeightToPixels(*v.Ht))
		}
	}
	if ws.SheetFormatPr != nil && ws.SheetFormatPr.DefaultRowHeight > 0 {
		return int(convertRowHeightToPixels(ws.SheetFormatPr.DefaultRowHeight))
	}
	// Optimization for when the row heights haven't changed.
	return int(defaultRowHeightPixels)
}

// GetRowHeight provides a function to get row height by given worksheet name
// and row number. For example, get the height of the first row in Sheet1:
//
//	height, err := f.GetRowHeight("Sheet1", 1)
func (f *File) GetRowHeight(sheet string, row int) (float64, error) {
	if row < 1 {
		return defaultRowHeight, newInvalidRowNumberError(row)
	}
	ht := defaultRowHeight
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return ht, err
	}
	if ws.SheetFormatPr != nil && ws.SheetFormatPr.CustomHeight {
		ht = ws.SheetFormatPr.DefaultRowHeight
	}
	if row > len(ws.SheetData.Row) {
		return ht, nil // it will be better to use 0, but we take care with BC
	}
	for _, v := range ws.SheetData.Row {
		if v.R == row && v.Ht != nil {
			return *v.Ht, nil
		}
	}
	// Optimization for when the row heights haven't changed.
	return ht, nil
}

// sharedStringsReader provides a function to get the pointer to the structure
// after deserialization of xl/sharedStrings.xml.
func (f *File) sharedStringsReader() (*xlsxSST, error) {
	var err error
	f.mu.Lock()
	defer f.mu.Unlock()
	relPath := f.getWorkbookRelsPath()
	if f.SharedStrings == nil {
		var sharedStrings xlsxSST
		ss := f.readXML(defaultXMLPathSharedStrings)
		if err = f.xmlNewDecoder(bytes.NewReader(namespaceStrictToTransitional(ss))).
			Decode(&sharedStrings); err != nil && err != io.EOF {
			return f.SharedStrings, err
		}
		if sharedStrings.Count == 0 {
			sharedStrings.Count = len(sharedStrings.SI)
		}
		if sharedStrings.UniqueCount == 0 {
			sharedStrings.UniqueCount = sharedStrings.Count
		}
		f.SharedStrings = &sharedStrings
		for i := range sharedStrings.SI {
			if sharedStrings.SI[i].T != nil {
				f.sharedStringsMap[sharedStrings.SI[i].T.Val] = i
			}
		}
		if err = f.addContentTypePart(0, "sharedStrings"); err != nil {
			return f.SharedStrings, err
		}
		rels, err := f.relsReader(relPath)
		if err != nil {
			return f.SharedStrings, err
		}
		for _, rel := range rels.Relationships {
			if rel.Target == "/xl/sharedStrings.xml" {
				return f.SharedStrings, nil
			}
		}
		// Update workbook.xml.rels
		f.addRels(relPath, SourceRelationshipSharedStrings, "/xl/sharedStrings.xml", "")
	}

	return f.SharedStrings, nil
}

// SetRowVisible provides a function to set visible of a single row by given
// worksheet name and Excel row number. For example, hide row 2 in Sheet1:
//
//	err := f.SetRowVisible("Sheet1", 2, false)
func (f *File) SetRowVisible(sheet string, row int, visible bool) error {
	if row < 1 {
		return newInvalidRowNumberError(row)
	}

	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	ws.prepareSheetXML(0, row)
	ws.SheetData.Row[row-1].Hidden = !visible
	return nil
}

// GetRowVisible provides a function to get visible of a single row by given
// worksheet name and Excel row number. For example, get visible state of row
// 2 in Sheet1:
//
//	visible, err := f.GetRowVisible("Sheet1", 2)
func (f *File) GetRowVisible(sheet string, row int) (bool, error) {
	if row < 1 {
		return false, newInvalidRowNumberError(row)
	}

	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return false, err
	}
	if row > len(ws.SheetData.Row) {
		return false, nil
	}
	return !ws.SheetData.Row[row-1].Hidden, nil
}

// SetRowOutlineLevel provides a function to set outline level number of a
// single row by given worksheet name and Excel row number. The value of
// parameter 'level' is 1-7. For example, outline row 2 in Sheet1 to level 1:
//
//	err := f.SetRowOutlineLevel("Sheet1", 2, 1)
func (f *File) SetRowOutlineLevel(sheet string, row int, level uint8) error {
	if row < 1 {
		return newInvalidRowNumberError(row)
	}
	if level > 7 || level < 1 {
		return ErrOutlineLevel
	}
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	ws.prepareSheetXML(0, row)
	ws.SheetData.Row[row-1].OutlineLevel = level
	return nil
}

// GetRowOutlineLevel provides a function to get outline level number of a
// single row by given worksheet name and Excel row number. For example, get
// outline number of row 2 in Sheet1:
//
//	level, err := f.GetRowOutlineLevel("Sheet1", 2)
func (f *File) GetRowOutlineLevel(sheet string, row int) (uint8, error) {
	if row < 1 {
		return 0, newInvalidRowNumberError(row)
	}
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return 0, err
	}
	if row > len(ws.SheetData.Row) {
		return 0, nil
	}
	return ws.SheetData.Row[row-1].OutlineLevel, nil
}

// RemoveRow provides a function to remove single row by given worksheet name
// and Excel row number. For example, remove row 3 in Sheet1:
//
//	err := f.RemoveRow("Sheet1", 3)
//
// Use this method with caution, which will affect changes in references such
// as formulas, charts, and so on. If there is any referenced value of the
// worksheet, it will cause a file error when you open it. The excelize only
// partially updates these references currently.
func (f *File) RemoveRow(sheet string, row int) error {
	if row < 1 {
		return newInvalidRowNumberError(row)
	}

	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	if row > len(ws.SheetData.Row) {
		return f.adjustHelper(sheet, rows, row, -1)
	}
	keep := 0
	for rowIdx := 0; rowIdx < len(ws.SheetData.Row); rowIdx++ {
		v := &ws.SheetData.Row[rowIdx]
		if v.R != row {
			ws.SheetData.Row[keep] = *v
			keep++
		}
	}
	ws.SheetData.Row = ws.SheetData.Row[:keep]
	return f.adjustHelper(sheet, rows, row, -1)
}

// InsertRows provides a function to insert new rows after the given Excel row
// number starting from 1 and number of rows. For example, create two rows
// before row 3 in Sheet1:
//
//	err := f.InsertRows("Sheet1", 3, 2)
//
// Use this method with caution, which will affect changes in references such
// as formulas, charts, and so on. If there is any referenced value of the
// worksheet, it will cause a file error when you open it. The excelize only
// partially updates these references currently.
func (f *File) InsertRows(sheet string, row, n int) error {
	if row < 1 {
		return newInvalidRowNumberError(row)
	}
	if row >= TotalRows || n >= TotalRows {
		return ErrMaxRows
	}
	if n < 1 {
		return ErrParameterInvalid
	}
	return f.adjustHelper(sheet, rows, row, n)
}

// DuplicateRow inserts a copy of specified row (by its Excel row number) below
//
//	err := f.DuplicateRow("Sheet1", 2)
//
// Use this method with caution, which will affect changes in references such
// as formulas, charts, and so on. If there is any referenced value of the
// worksheet, it will cause a file error when you open it. The excelize only
// partially updates these references currently.
func (f *File) DuplicateRow(sheet string, row int) error {
	return f.DuplicateRowTo(sheet, row, row+1)
}

// DuplicateRowTo inserts a copy of specified row by it Excel number
// to specified row position moving down exists rows after target position
//
//	err := f.DuplicateRowTo("Sheet1", 2, 7)
//
// Use this method with caution, which will affect changes in references such
// as formulas, charts, and so on. If there is any referenced value of the
// worksheet, it will cause a file error when you open it. The excelize only
// partially updates these references currently.
func (f *File) DuplicateRowTo(sheet string, row, row2 int) error {
	if row < 1 {
		return newInvalidRowNumberError(row)
	}

	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}

	if row2 < 1 || row == row2 {
		return nil
	}

	var ok bool
	var rowCopy xlsxRow

	for i, r := range ws.SheetData.Row {
		if r.R == row {
			rowCopy = deepcopy.Copy(ws.SheetData.Row[i]).(xlsxRow)
			ok = true
			break
		}
	}

	if err := f.adjustHelper(sheet, rows, row2, 1); err != nil {
		return err
	}

	if !ok {
		return nil
	}

	idx2 := -1
	for i, r := range ws.SheetData.Row {
		if r.R == row2 {
			idx2 = i
			break
		}
	}
	if idx2 == -1 && len(ws.SheetData.Row) >= row2 {
		return nil
	}

	rowCopy.C = append(make([]xlsxC, 0, len(rowCopy.C)), rowCopy.C...)
	rowCopy.adjustSingleRowDimensions(row2 - row)
	_ = f.adjustSingleRowFormulas(sheet, sheet, &rowCopy, row, row2-row, true)

	if idx2 != -1 {
		ws.SheetData.Row[idx2] = rowCopy
	} else {
		ws.SheetData.Row = append(ws.SheetData.Row, rowCopy)
	}
	return f.duplicateMergeCells(sheet, ws, row, row2)
}

// duplicateMergeCells merge cells in the destination row if there are single
// row merged cells in the copied row.
func (f *File) duplicateMergeCells(sheet string, ws *xlsxWorksheet, row, row2 int) error {
	if ws.MergeCells == nil {
		return nil
	}
	if row > row2 {
		row++
	}
	for _, rng := range ws.MergeCells.Cells {
		coordinates, err := rangeRefToCoordinates(rng.Ref)
		if err != nil {
			return err
		}
		if coordinates[1] < row2 && row2 < coordinates[3] {
			return nil
		}
	}
	for i := 0; i < len(ws.MergeCells.Cells); i++ {
		mergedCells := ws.MergeCells.Cells[i]
		coordinates, _ := rangeRefToCoordinates(mergedCells.Ref)
		x1, y1, x2, y2 := coordinates[0], coordinates[1], coordinates[2], coordinates[3]
		if y1 == y2 && y1 == row {
			from, _ := CoordinatesToCellName(x1, row2)
			to, _ := CoordinatesToCellName(x2, row2)
			if err := f.MergeCell(sheet, from, to); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkRow provides a function to check and fill each column element for all
// rows and make that is continuous in a worksheet of XML. For example:
//
//	<row r="15">
//	    <c r="A15" s="2" />
//	    <c r="B15" s="2" />
//	    <c r="F15" s="1" />
//	    <c r="G15" s="1" />
//	</row>
//
// in this case, we should to change it to
//
//	<row r="15">
//	    <c r="A15" s="2" />
//	    <c r="B15" s="2" />
//	    <c r="C15" s="2" />
//	    <c r="D15" s="2" />
//	    <c r="E15" s="2" />
//	    <c r="F15" s="1" />
//	    <c r="G15" s="1" />
//	</row>
//
// Notice: this method could be very slow for large spreadsheets (more than
// 3000 rows one sheet).
func (ws *xlsxWorksheet) checkRow() error {
	for rowIdx := range ws.SheetData.Row {
		rowData := &ws.SheetData.Row[rowIdx]

		colCount := len(rowData.C)
		if colCount == 0 {
			continue
		}
		// check and fill the cell without r attribute in a row element
		rCount := 0
		for idx, cell := range rowData.C {
			rCount++
			if cell.R != "" {
				lastR, _, err := CellNameToCoordinates(cell.R)
				if err != nil {
					return err
				}
				if lastR > rCount {
					rCount = lastR
				}
				continue
			}
			rowData.C[idx].R, _ = CoordinatesToCellName(rCount, rowIdx+1)
		}
		lastCol, _, err := CellNameToCoordinates(rowData.C[colCount-1].R)
		if err != nil {
			return err
		}

		if colCount < lastCol {
			sourceList := rowData.C
			targetList := make([]xlsxC, 0, lastCol)

			rowData.C = ws.SheetData.Row[rowIdx].C[:0]

			for colIdx := 0; colIdx < lastCol; colIdx++ {
				cellName, err := CoordinatesToCellName(colIdx+1, rowIdx+1)
				if err != nil {
					return err
				}
				targetList = append(targetList, xlsxC{R: cellName})
			}

			rowData.C = targetList

			for colIdx := range sourceList {
				colData := &sourceList[colIdx]
				colNum, _, err := CellNameToCoordinates(colData.R)
				if err != nil {
					return err
				}
				ws.SheetData.Row[rowIdx].C[colNum-1] = *colData
			}
		}
	}
	return nil
}

// hasAttr determine if row non-default attributes.
func (r *xlsxRow) hasAttr() bool {
	return r.Spans != "" || r.S != 0 || r.CustomFormat || r.Ht != nil ||
		r.Hidden || r.CustomHeight || r.OutlineLevel != 0 || r.Collapsed ||
		r.ThickTop || r.ThickBot || r.Ph
}

// SetRowStyle provides a function to set the style of rows by given worksheet
// name, row range, and style ID. Note that this will overwrite the existing
// styles for the rows, it won't append or merge style with existing styles.
//
// For example set style of row 1 on Sheet1:
//
//	err := f.SetRowStyle("Sheet1", 1, 1, styleID)
//
// Set style of rows 1 to 10 on Sheet1:
//
//	err := f.SetRowStyle("Sheet1", 1, 10, styleID)
func (f *File) SetRowStyle(sheet string, start, end, styleID int) error {
	if end < start {
		start, end = end, start
	}
	if start < 1 {
		return newInvalidRowNumberError(start)
	}
	if end > TotalRows {
		return ErrMaxRows
	}
	s, err := f.stylesReader()
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if styleID < 0 || s.CellXfs == nil || len(s.CellXfs.Xf) <= styleID {
		return newInvalidStyleID(styleID)
	}
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	ws.prepareSheetXML(0, end)
	for row := start - 1; row < end; row++ {
		ws.SheetData.Row[row].S = styleID
		ws.SheetData.Row[row].CustomFormat = true
		for i := range ws.SheetData.Row[row].C {
			if _, rowNum, err := CellNameToCoordinates(ws.SheetData.Row[row].C[i].R); err == nil && rowNum-1 == row {
				ws.SheetData.Row[row].C[i].S = styleID
			}
		}
	}
	return nil
}

// convertRowHeightToPixels provides a function to convert the height of a
// cell from user's units to pixels. If the height hasn't been set by the user
// we use the default value. If the row is hidden it has a value of zero.
func convertRowHeightToPixels(height float64) float64 {
	if height == 0 {
		return 0
	}
	return math.Ceil(4.0 / 3.4 * height)
}
