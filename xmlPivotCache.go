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

import "encoding/xml"

// xlsxPivotCacheDefinition represents the pivotCacheDefinition part. This part
// defines each field in the source data, including the name, the string
// resources of the instance data (for shared items), and information about
// the type of data that appears in the field.
type xlsxPivotCacheDefinition struct {
	XMLName               xml.Name               `xml:"http://schemas.openxmlformats.org/spreadsheetml/2006/main pivotCacheDefinition"`
	RID                   string                 `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr,omitempty"`
	Invalid               bool                   `xml:"invalid,attr,omitempty"`
	SaveData              bool                   `xml:"saveData,attr"`
	RefreshOnLoad         bool                   `xml:"refreshOnLoad,attr,omitempty"`
	OptimizeMemory        bool                   `xml:"optimizeMemory,attr,omitempty"`
	EnableRefresh         bool                   `xml:"enableRefresh,attr,omitempty"`
	RefreshedBy           string                 `xml:"refreshedBy,attr,omitempty"`
	RefreshedDate         float64                `xml:"refreshedDate,attr,omitempty"`
	RefreshedDateIso      float64                `xml:"refreshedDateIso,attr,omitempty"`
	BackgroundQuery       bool                   `xml:"backgroundQuery,attr"`
	MissingItemsLimit     int                    `xml:"missingItemsLimit,attr,omitempty"`
	CreatedVersion        int                    `xml:"createdVersion,attr,omitempty"`
	RefreshedVersion      int                    `xml:"refreshedVersion,attr,omitempty"`
	MinRefreshableVersion int                    `xml:"minRefreshableVersion,attr,omitempty"`
	RecordCount           int                    `xml:"recordCount,attr,omitempty"`
	UpgradeOnRefresh      bool                   `xml:"upgradeOnRefresh,attr,omitempty"`
	TupleCacheAttr        bool                   `xml:"tupleCache,attr,omitempty"`
	SupportSubquery       bool                   `xml:"supportSubquery,attr,omitempty"`
	SupportAdvancedDrill  bool                   `xml:"supportAdvancedDrill,attr,omitempty"`
	CacheSource           *xlsxCacheSource       `xml:"cacheSource"`
	CacheFields           *xlsxCacheFields       `xml:"cacheFields"`
	CacheHierarchies      *xlsxCacheHierarchies  `xml:"cacheHierarchies"`
	Kpis                  *xlsxKpis              `xml:"kpis"`
	TupleCache            *xlsxTupleCache        `xml:"tupleCache"`
	CalculatedItems       *xlsxCalculatedItems   `xml:"calculatedItems"`
	CalculatedMembers     *xlsxCalculatedMembers `xml:"calculatedMembers"`
	Dimensions            *xlsxDimensions        `xml:"dimensions"`
	MeasureGroups         *xlsxMeasureGroups     `xml:"measureGroups"`
	Maps                  *xlsxMaps              `xml:"maps"`
	ExtLst                *xlsxExtLst            `xml:"extLst"`
}

// xlsxCacheSource represents the description of data source whose data is
// stored in the pivot cache. The data source refers to the underlying rows or
// database records that provide the data for a PivotTable. You can create a
// PivotTable report from a SpreadsheetML table, an external database
// (including OLAP cubes), multiple SpreadsheetML worksheets, or another
// PivotTable.
type xlsxCacheSource struct {
	Type            string               `xml:"type,attr"`
	ConnectionID    int                  `xml:"connectionId,attr,omitempty"`
	WorksheetSource *xlsxWorksheetSource `xml:"worksheetSource"`
	Consolidation   *xlsxConsolidation   `xml:"consolidation"`
	ExtLst          *xlsxExtLst          `xml:"extLst"`
}

// xlsxWorksheetSource represents the location of the source of the data that
// is stored in the cache.
type xlsxWorksheetSource struct {
	RID   string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr,omitempty"`
	Ref   string `xml:"ref,attr,omitempty"`
	Name  string `xml:"name,attr,omitempty"`
	Sheet string `xml:"sheet,attr,omitempty"`
}

// xlsxConsolidation represents the description of the PivotCache source using
// multiple consolidation ranges. This element is used when the source of the
// PivotTable is a collection of ranges in the workbook. The ranges are
// specified in the rangeSets collection. The logic for how the application
// consolidates the data in the ranges is application- defined.
type xlsxConsolidation struct{}

// xlsxCacheFields represents the collection of field definitions in the
// source data.
type xlsxCacheFields struct {
	Count      int               `xml:"count,attr"`
	CacheField []*xlsxCacheField `xml:"cacheField"`
}

// xlsxCacheField represent a single field in the PivotCache. This definition
// contains information about the field, such as its source, data type, and
// location within a level or hierarchy. The sharedItems element stores
// additional information about the data in this field. If there are no shared
// items, then values are stored directly in the pivotCacheRecords part.
type xlsxCacheField struct {
	Name                string           `xml:"name,attr"`
	Caption             string           `xml:"caption,attr,omitempty"`
	PropertyName        string           `xml:"propertyName,attr,omitempty"`
	ServerField         bool             `xml:"serverField,attr,omitempty"`
	UniqueList          bool             `xml:"uniqueList,attr,omitempty"`
	NumFmtID            int              `xml:"numFmtId,attr"`
	Formula             string           `xml:"formula,attr,omitempty"`
	SQLType             int              `xml:"sqlType,attr,omitempty"`
	Hierarchy           int              `xml:"hierarchy,attr,omitempty"`
	Level               int              `xml:"level,attr,omitempty"`
	DatabaseField       bool             `xml:"databaseField,attr,omitempty"`
	MappingCount        int              `xml:"mappingCount,attr,omitempty"`
	MemberPropertyField bool             `xml:"memberPropertyField,attr,omitempty"`
	SharedItems         *xlsxSharedItems `xml:"sharedItems"`
	FieldGroup          *xlsxFieldGroup  `xml:"fieldGroup"`
	MpMap               *xlsxX           `xml:"mpMap"`
	ExtLst              *xlsxExtLst      `xml:"extLst"`
}

// xlsxSharedItems represents the collection of unique items for a field in
// the PivotCacheDefinition. The sharedItems complex type stores data type and
// formatting information about the data in a field. Items in the
// PivotCacheDefinition can be shared in order to reduce the redundancy of
// those values that are referenced in multiple places across all the
// PivotTable parts.
type xlsxSharedItems struct {
	ContainsSemiMixedTypes bool           `xml:"containsSemiMixedTypes,attr,omitempty"`
	ContainsNonDate        bool           `xml:"containsNonDate,attr,omitempty"`
	ContainsDate           bool           `xml:"containsDate,attr,omitempty"`
	ContainsString         bool           `xml:"containsString,attr,omitempty"`
	ContainsBlank          bool           `xml:"containsBlank,attr,omitempty"`
	ContainsMixedTypes     bool           `xml:"containsMixedTypes,attr,omitempty"`
	ContainsNumber         bool           `xml:"containsNumber,attr,omitempty"`
	ContainsInteger        bool           `xml:"containsInteger,attr,omitempty"`
	MinValue               float64        `xml:"minValue,attr,omitempty"`
	MaxValue               float64        `xml:"maxValue,attr,omitempty"`
	MinDate                string         `xml:"minDate,attr,omitempty"`
	MaxDate                string         `xml:"maxDate,attr,omitempty"`
	Count                  int            `xml:"count,attr"`
	LongText               bool           `xml:"longText,attr,omitempty"`
	M                      []xlsxMissing  `xml:"m"`
	N                      []xlsxNumber   `xml:"n"`
	B                      []xlsxBoolean  `xml:"b"`
	E                      []xlsxError    `xml:"e"`
	S                      []xlsxString   `xml:"s"`
	D                      []xlsxDateTime `xml:"d"`
}

// xlsxMissing represents a value that was not specified.
type xlsxMissing struct{}

// xlsxNumber represents a numeric value in the PivotTable.
type xlsxNumber struct {
	V    float64     `xml:"v,attr"`
	U    bool        `xml:"u,attr,omitempty"`
	F    bool        `xml:"f,attr,omitempty"`
	C    string      `xml:"c,attr,omitempty"`
	Cp   int         `xml:"cp,attr,omitempty"`
	In   int         `xml:"in,attr,omitempty"`
	Bc   string      `xml:"bc,attr,omitempty"`
	Fc   string      `xml:"fc,attr,omitempty"`
	I    bool        `xml:"i,attr,omitempty"`
	Un   bool        `xml:"un,attr,omitempty"`
	St   bool        `xml:"st,attr,omitempty"`
	B    bool        `xml:"b,attr,omitempty"`
	Tpls *xlsxTuples `xml:"tpls"`
	X    *attrValInt `xml:"x"`
}

// xlsxTuples represents members for the OLAP sheet data entry, also known as
// a tuple.
type xlsxTuples struct{}

// xlsxBoolean represents a boolean value for an item in the PivotTable.
type xlsxBoolean struct{}

// xlsxError represents an error value. The use of this item indicates that an
// error value is present in the PivotTable source. The error is recorded in
// the value attribute.
type xlsxError struct{}

// xlsxString represents a character value in a PivotTable.
type xlsxString struct {
	V    string      `xml:"v,attr"`
	U    bool        `xml:"u,attr,omitempty"`
	F    bool        `xml:"f,attr,omitempty"`
	C    string      `xml:"c,attr,omitempty"`
	Cp   int         `xml:"cp,attr,omitempty"`
	In   int         `xml:"in,attr,omitempty"`
	Bc   string      `xml:"bc,attr,omitempty"`
	Fc   string      `xml:"fc,attr,omitempty"`
	I    bool        `xml:"i,attr,omitempty"`
	Un   bool        `xml:"un,attr,omitempty"`
	St   bool        `xml:"st,attr,omitempty"`
	B    bool        `xml:"b,attr,omitempty"`
	Tpls *xlsxTuples `xml:"tpls"`
	X    *attrValInt `xml:"x"`
}

// xlsxDateTime represents a date-time value in the PivotTable.
type xlsxDateTime struct{}

// xlsxFieldGroup represents the collection of properties for a field group.
type xlsxFieldGroup struct{}

// xlsxCacheHierarchies represents the collection of OLAP hierarchies in the
// PivotCache.
type xlsxCacheHierarchies struct{}

// xlsxKpis represents the collection of Key Performance Indicators (KPIs)
// defined on the OLAP server and stored in the PivotCache.
type xlsxKpis struct{}

// xlsxTupleCache represents the cache of OLAP sheet data members, or tuples.
type xlsxTupleCache struct{}

// xlsxCalculatedItems represents the collection of calculated items.
type xlsxCalculatedItems struct{}

// xlsxCalculatedMembers represents the collection of calculated members in an
// OLAP PivotTable.
type xlsxCalculatedMembers struct{}

// xlsxDimensions represents the collection of PivotTable OLAP dimensions.
type xlsxDimensions struct{}

// xlsxMeasureGroups represents the collection of PivotTable OLAP measure
// groups.
type xlsxMeasureGroups struct{}

// xlsxMaps represents the PivotTable OLAP measure group - Dimension maps.
type xlsxMaps struct{}

// xlsxX14PivotCacheDefinition specifies the extended properties of a pivot
// table cache definition.
type xlsxX14PivotCacheDefinition struct {
	XMLName      xml.Name `xml:"x14:pivotCacheDefinition"`
	PivotCacheID int      `xml:"pivotCacheId,attr"`
}

// decodeX14PivotCacheDefinition defines the structure used to parse the
// x14:pivotCacheDefinition element of a pivot table cache.
type decodeX14PivotCacheDefinition struct {
	XMLName      xml.Name `xml:"pivotCacheDefinition"`
	PivotCacheID int      `xml:"pivotCacheId,attr"`
}
