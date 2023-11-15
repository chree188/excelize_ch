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
	"fmt"
	"math"
	"strings"
	"unicode/utf16"
)

// DataValidationType defined the type of data validation.
type DataValidationType int

// Data validation types.
const (
	_ DataValidationType = iota
	DataValidationTypeNone
	DataValidationTypeCustom
	DataValidationTypeDate
	DataValidationTypeDecimal
	DataValidationTypeList
	DataValidationTypeTextLength
	DataValidationTypeTime
	DataValidationTypeWhole
)

// DataValidationErrorStyle defined the style of data validation error alert.
type DataValidationErrorStyle int

// Data validation error styles.
const (
	_ DataValidationErrorStyle = iota
	DataValidationErrorStyleStop
	DataValidationErrorStyleWarning
	DataValidationErrorStyleInformation
)

// Data validation error styles.
const (
	styleStop        = "stop"
	styleWarning     = "warning"
	styleInformation = "information"
)

// DataValidationOperator operator enum.
type DataValidationOperator int

// Data validation operators.
const (
	_ DataValidationOperator = iota
	DataValidationOperatorBetween
	DataValidationOperatorEqual
	DataValidationOperatorGreaterThan
	DataValidationOperatorGreaterThanOrEqual
	DataValidationOperatorLessThan
	DataValidationOperatorLessThanOrEqual
	DataValidationOperatorNotBetween
	DataValidationOperatorNotEqual
)

var (
	// formulaEscaper mimics the Excel escaping rules for data validation,
	// which converts `"` to `""` instead of `&quot;`.
	formulaEscaper = strings.NewReplacer(
		`&`, `&amp;`,
		`<`, `&lt;`,
		`>`, `&gt;`,
	)
	formulaUnescaper = strings.NewReplacer(
		`&amp;`, `&`,
		`&lt;`, `<`,
		`&gt;`, `>`,
	)
	// dataValidationTypeMap defined supported data validation types.
	dataValidationTypeMap = map[DataValidationType]string{
		DataValidationTypeNone:       "none",
		DataValidationTypeCustom:     "custom",
		DataValidationTypeDate:       "date",
		DataValidationTypeDecimal:    "decimal",
		DataValidationTypeList:       "list",
		DataValidationTypeTextLength: "textLength",
		DataValidationTypeTime:       "time",
		DataValidationTypeWhole:      "whole",
	}
	// dataValidationOperatorMap defined supported data validation operators.
	dataValidationOperatorMap = map[DataValidationOperator]string{
		DataValidationOperatorBetween:            "between",
		DataValidationOperatorEqual:              "equal",
		DataValidationOperatorGreaterThan:        "greaterThan",
		DataValidationOperatorGreaterThanOrEqual: "greaterThanOrEqual",
		DataValidationOperatorLessThan:           "lessThan",
		DataValidationOperatorLessThanOrEqual:    "lessThanOrEqual",
		DataValidationOperatorNotBetween:         "notBetween",
		DataValidationOperatorNotEqual:           "notEqual",
	}
)

// NewDataValidation return data validation struct.
func NewDataValidation(allowBlank bool) *DataValidation {
	return &DataValidation{
		AllowBlank:       allowBlank,
		ShowErrorMessage: false,
		ShowInputMessage: false,
	}
}

// SetError set error notice.
func (dv *DataValidation) SetError(style DataValidationErrorStyle, title, msg string) {
	dv.Error = &msg
	dv.ErrorTitle = &title
	strStyle := styleStop
	switch style {
	case DataValidationErrorStyleStop:
		strStyle = styleStop
	case DataValidationErrorStyleWarning:
		strStyle = styleWarning
	case DataValidationErrorStyleInformation:
		strStyle = styleInformation

	}
	dv.ShowErrorMessage = true
	dv.ErrorStyle = &strStyle
}

// SetInput set prompt notice.
func (dv *DataValidation) SetInput(title, msg string) {
	dv.ShowInputMessage = true
	dv.PromptTitle = &title
	dv.Prompt = &msg
}

// SetDropList data validation list. If you type the items into the data
// validation dialog box (a delimited list), the limit is 255 characters,
// including the separators. If your data validation list source formula is
// over the maximum length limit, please set the allowed values in the
// worksheet cells, and use the SetSqrefDropList function to set the reference
// for their cells.
func (dv *DataValidation) SetDropList(keys []string) error {
	formula := strings.Join(keys, ",")
	if MaxFieldLength < len(utf16.Encode([]rune(formula))) {
		return ErrDataValidationFormulaLength
	}
	dv.Type = dataValidationTypeMap[DataValidationTypeList]
	if strings.HasPrefix(formula, "=") {
		dv.Formula1 = formulaEscaper.Replace(formula)
		return nil
	}
	dv.Formula1 = fmt.Sprintf(`"%s"`, strings.NewReplacer(`"`, `""`).Replace(formulaEscaper.Replace(formula)))
	return nil
}

// SetRange provides function to set data validation range in drop list, only
// accepts int, float64, string or []string data type formula argument.
func (dv *DataValidation) SetRange(f1, f2 interface{}, t DataValidationType, o DataValidationOperator) error {
	genFormula := func(val interface{}) (string, error) {
		var formula string
		switch v := val.(type) {
		case int:
			formula = fmt.Sprintf("%d", v)
		case float64:
			if math.Abs(v) > math.MaxFloat32 {
				return formula, ErrDataValidationRange
			}
			formula = fmt.Sprintf("%.17g", v)
		case string:
			formula = v
		default:
			return formula, ErrParameterInvalid
		}
		return formula, nil
	}
	formula1, err := genFormula(f1)
	if err != nil {
		return err
	}
	formula2, err := genFormula(f2)
	if err != nil {
		return err
	}
	dv.Formula1, dv.Formula2 = formula1, formula2
	dv.Type = dataValidationTypeMap[t]
	dv.Operator = dataValidationOperatorMap[o]
	return err
}

// SetSqrefDropList provides set data validation on a range with source
// reference range of the worksheet by given data validation object and
// worksheet name. The data validation object can be created by
// NewDataValidation function. There are limits to the number of items that
// will show in a data validation drop down list: The list can show up to show
// 32768 items from a list on the worksheet. If you need more items than that,
// you could create a dependent drop down list, broken down by category. For
// example, set data validation on Sheet1!A7:B8 with validation criteria source
// Sheet1!E1:E3 settings, create in-cell dropdown by allowing list source:
//
//	dv := excelize.NewDataValidation(true)
//	dv.Sqref = "A7:B8"
//	dv.SetSqrefDropList("$E$1:$E$3")
//	err := f.AddDataValidation("Sheet1", dv)
func (dv *DataValidation) SetSqrefDropList(sqref string) {
	dv.Formula1 = sqref
	dv.Type = dataValidationTypeMap[DataValidationTypeList]
}

// SetSqref provides function to set data validation range in drop list.
func (dv *DataValidation) SetSqref(sqref string) {
	if dv.Sqref == "" {
		dv.Sqref = sqref
		return
	}
	dv.Sqref = fmt.Sprintf("%s %s", dv.Sqref, sqref)
}

// AddDataValidation provides set data validation on a range of the worksheet
// by given data validation object and worksheet name. The data validation
// object can be created by NewDataValidation function.
//
// Example 1, set data validation on Sheet1!A1:B2 with validation criteria
// settings, show error alert after invalid data is entered with "Stop" style
// and custom title "error body":
//
//	dv := excelize.NewDataValidation(true)
//	dv.Sqref = "A1:B2"
//	dv.SetRange(10, 20, excelize.DataValidationTypeWhole, excelize.DataValidationOperatorBetween)
//	dv.SetError(excelize.DataValidationErrorStyleStop, "error title", "error body")
//	err := f.AddDataValidation("Sheet1", dv)
//
// Example 2, set data validation on Sheet1!A3:B4 with validation criteria
// settings, and show input message when cell is selected:
//
//	dv = excelize.NewDataValidation(true)
//	dv.Sqref = "A3:B4"
//	dv.SetRange(10, 20, excelize.DataValidationTypeWhole, excelize.DataValidationOperatorGreaterThan)
//	dv.SetInput("input title", "input body")
//	err = f.AddDataValidation("Sheet1", dv)
//
// Example 3, set data validation on Sheet1!A5:B6 with validation criteria
// settings, create in-cell dropdown by allowing list source:
//
//	dv = excelize.NewDataValidation(true)
//	dv.Sqref = "A5:B6"
//	dv.SetDropList([]string{"1", "2", "3"})
//	err = f.AddDataValidation("Sheet1", dv)
func (f *File) AddDataValidation(sheet string, dv *DataValidation) error {
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	if nil == ws.DataValidations {
		ws.DataValidations = new(xlsxDataValidations)
	}
	dataValidation := &xlsxDataValidation{
		AllowBlank:       dv.AllowBlank,
		Error:            dv.Error,
		ErrorStyle:       dv.ErrorStyle,
		ErrorTitle:       dv.ErrorTitle,
		Operator:         dv.Operator,
		Prompt:           dv.Prompt,
		PromptTitle:      dv.PromptTitle,
		ShowDropDown:     dv.ShowDropDown,
		ShowErrorMessage: dv.ShowErrorMessage,
		ShowInputMessage: dv.ShowInputMessage,
		Sqref:            dv.Sqref,
		Type:             dv.Type,
	}
	if dv.Formula1 != "" {
		dataValidation.Formula1 = &xlsxInnerXML{Content: dv.Formula1}
	}
	if dv.Formula2 != "" {
		dataValidation.Formula2 = &xlsxInnerXML{Content: dv.Formula2}
	}
	ws.DataValidations.DataValidation = append(ws.DataValidations.DataValidation, dataValidation)
	ws.DataValidations.Count = len(ws.DataValidations.DataValidation)
	return err
}

// GetDataValidations returns data validations list by given worksheet name.
func (f *File) GetDataValidations(sheet string) ([]*DataValidation, error) {
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return nil, err
	}
	if ws.DataValidations == nil || len(ws.DataValidations.DataValidation) == 0 {
		return nil, err
	}
	var dvs []*DataValidation
	for _, dv := range ws.DataValidations.DataValidation {
		if dv != nil {
			dataValidation := &DataValidation{
				AllowBlank:       dv.AllowBlank,
				Error:            dv.Error,
				ErrorStyle:       dv.ErrorStyle,
				ErrorTitle:       dv.ErrorTitle,
				Operator:         dv.Operator,
				Prompt:           dv.Prompt,
				PromptTitle:      dv.PromptTitle,
				ShowDropDown:     dv.ShowDropDown,
				ShowErrorMessage: dv.ShowErrorMessage,
				ShowInputMessage: dv.ShowInputMessage,
				Sqref:            dv.Sqref,
				Type:             dv.Type,
			}
			if dv.Formula1 != nil {
				dataValidation.Formula1 = unescapeDataValidationFormula(dv.Formula1.Content)
			}
			if dv.Formula2 != nil {
				dataValidation.Formula2 = unescapeDataValidationFormula(dv.Formula2.Content)
			}
			dvs = append(dvs, dataValidation)
		}
	}
	return dvs, err
}

// DeleteDataValidation delete data validation by given worksheet name and
// reference sequence. All data validations in the worksheet will be deleted
// if not specify reference sequence parameter.
func (f *File) DeleteDataValidation(sheet string, sqref ...string) error {
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	if ws.DataValidations == nil {
		return nil
	}
	if sqref == nil {
		ws.DataValidations = nil
		return nil
	}
	delCells, err := f.flatSqref(sqref[0])
	if err != nil {
		return err
	}
	dv := ws.DataValidations
	for i := 0; i < len(dv.DataValidation); i++ {
		var applySqref []string
		colCells, err := f.flatSqref(dv.DataValidation[i].Sqref)
		if err != nil {
			return err
		}
		for col, cells := range delCells {
			for _, cell := range cells {
				idx := inCoordinates(colCells[col], cell)
				if idx != -1 {
					colCells[col] = append(colCells[col][:idx], colCells[col][idx+1:]...)
				}
			}
		}
		for _, col := range colCells {
			applySqref = append(applySqref, f.squashSqref(col)...)
		}
		dv.DataValidation[i].Sqref = strings.Join(applySqref, " ")
		if len(applySqref) == 0 {
			dv.DataValidation = append(dv.DataValidation[:i], dv.DataValidation[i+1:]...)
			i--
		}
	}
	dv.Count = len(dv.DataValidation)
	if dv.Count == 0 {
		ws.DataValidations = nil
	}
	return nil
}

// squashSqref generates cell reference sequence by given cells coordinates list.
func (f *File) squashSqref(cells [][]int) []string {
	if len(cells) == 1 {
		cell, _ := CoordinatesToCellName(cells[0][0], cells[0][1])
		return []string{cell}
	} else if len(cells) == 0 {
		return []string{}
	}
	var refs []string
	l, r := 0, 0
	for i := 1; i < len(cells); i++ {
		if cells[i][0] == cells[r][0] && cells[i][1]-cells[r][1] > 1 {
			ref, _ := f.coordinatesToRangeRef(append(cells[l], cells[r]...))
			if l == r {
				ref, _ = CoordinatesToCellName(cells[l][0], cells[l][1])
			}
			refs = append(refs, ref)
			l, r = i, i
		} else {
			r++
		}
	}
	ref, _ := f.coordinatesToRangeRef(append(cells[l], cells[r]...))
	if l == r {
		ref, _ = CoordinatesToCellName(cells[l][0], cells[l][1])
	}
	return append(refs, ref)
}

// unescapeDataValidationFormula returns unescaped data validation formula.
func unescapeDataValidationFormula(val string) string {
	if strings.HasPrefix(val, "\"") { // Text detection
		return strings.NewReplacer(`""`, `"`).Replace(formulaUnescaper.Replace(val))
	}
	return formulaUnescaper.Replace(val)
}
