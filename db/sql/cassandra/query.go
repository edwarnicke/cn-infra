package cassandra

import (
	"bytes"
	"errors"
	"fmt"
	r "reflect"
	"strings"

	"github.com/ligato/cn-infra/db/sql"
	"github.com/ligato/cn-infra/utils/structs"
)

// PutExpToString converts expression to string & slice of bindings
func PutExpToString(whereCondition sql.Expression, entity interface{}) (sqlStr string, bindings []interface{},
	err error) {

	whereCondtionStr := &toStringVisitor{entity: entity}
	whereCondition.Accept(whereCondtionStr)

	statement, _, err := updateSetExpToString(sql.EntityTableName(entity), /*TODO extract method / make customizable*/
		entity                                                             /*, TODO TTL*/)
	if err != nil {
		return "", nil, err
	}

	bindings = structs.ListExportedFieldsPtrs(entity, cqlExported)
	whereBinding := whereCondtionStr.Binding()
	if whereBinding != nil {
		bindings = append(bindings, whereBinding...)
	}

	return strings.Trim(statement+" WHERE"+whereCondtionStr.String(), " "), bindings, nil
}

// SelectExpToString converts expression to string & slice of bindings
func SelectExpToString(fromWhere sql.Expression) (sqlStr string, bindings []interface{},
	err error) {

	findEntity := &findEntityVisitor{}
	fromWhere.Accept(findEntity)

	fromWhereStr := &toStringVisitor{entity: findEntity.entity}
	fromWhere.Accept(fromWhereStr)

	fieldsStr := selectFields(findEntity.entity)
	if err != nil {
		return "", nil, err
	}
	fromWhereBindings := fromWhereStr.Binding()

	return "SELECT " + fieldsStr + fromWhereStr.String(), fromWhereBindings, nil
}

// ExpToString converts expression to string & slice of bindings
func ExpToString(exp sql.Expression) (sql string, bindings []interface{}, err error) {
	findEntity := &findEntityVisitor{}
	exp.Accept(findEntity)

	stringer := &toStringVisitor{entity: findEntity.entity}
	exp.Accept(stringer)

	return stringer.String(), stringer.Binding(), stringer.lastError
}

type toStringVisitor struct {
	entity    interface{}
	generated bytes.Buffer
	binding   []interface{}
	lastError error
}

// String converts generated byte Buffer to string
func (visitor *toStringVisitor) String() string {
	return visitor.generated.String()
}

// Binding is a getter...
func (visitor *toStringVisitor) Binding() []interface{} {
	return visitor.binding
}

// VisitPrefixedExp generates part of SQL expression
func (visitor *toStringVisitor) VisitPrefixedExp(exp *sql.PrefixedExp) {
	visitor.generated.WriteString(" ")
	visitor.generated.WriteString(exp.Prefix)
	if exp.Prefix == "FROM" {
		visitor.generated.WriteString(" ")
		visitor.generated.WriteString(sql.EntityTableName(visitor.entity))
	}
	if exp.AfterPrefix != nil {
		exp.AfterPrefix.Accept(visitor)
	}
	visitor.generated.WriteString(exp.Suffix)

	if exp.Prefix != "FROM" && exp.Binding != nil && len(exp.Binding) > 0 {
		if visitor.binding != nil {
			visitor.binding = append(visitor.binding, exp.Binding)
		} else {
			visitor.binding = exp.Binding
		}
	}
}

// VisitPrefixedExp generates part of SQL expression
func (visitor *toStringVisitor) VisitFieldExpression(exp *sql.FieldExpression) {
	if visitor.entity == nil {
		visitor.lastError = errors.New("not found entity")
	} else {
		field, found := structs.FindField(exp.PointerToAField, visitor.entity)
		if !found {
			visitor.lastError = errors.New("not found field in entity")
			return
		}
		fieldName, found := fieldName(field)
		if !found {
			visitor.lastError = errors.New("not exported field in entity")
			return
		}
		visitor.generated.WriteString(" ")
		visitor.generated.WriteString(fieldName)

		if exp.AfterField != nil {
			exp.AfterField.Accept(visitor)
		}
	}
}

// cqlExported checks the cql tag in StructField and parses the field name
func cqlExported(field *r.StructField) (exported bool) {
	cql := field.Tag.Get("cql")
	if len(cql) > 0 {
		if cql == "-" {
			return false
		}
		return true
	}
	return true
}

// cqlExportedWithFieldName checks the cql tag in StructField and parses the field name
func cqlExportedWithFieldName(field *r.StructField) (fieldName string, exported bool) {
	cql := field.Tag.Get("cql")
	if len(cql) > 0 {
		if cql == "-" {
			return cql, false
		}
		return cql, true
	}
	return field.Name, true
}

func fieldName(field *r.StructField) (name string, exported bool) {
	structExported := structs.FieldExported(field)
	if !structExported {
		return field.Name, structExported
	}

	return cqlExportedWithFieldName(field)
}

// selectFields generates comma separated field names string
func selectFields(val interface{} /*, opts Options*/) (statement string) {
	fields := structs.ListExportedFields(val, cqlExported)
	ret := bytes.Buffer{}
	first := true
	for _, field := range fields {
		fieldName, exported := fieldName(field)
		if exported {
			if first {
				first = false
			} else {
				ret.WriteString(", ")
			}

			ret.WriteString(fieldName)
		}
	}

	return ret.String()
}

// SliceOfFields generates slice of translated (cql tag) field names
func sliceOfFieldNames(val interface{} /*, opts Options*/) (fieldNames []string) {
	fields := structs.ListExportedFields(val)
	fieldNames = []string{}
	for _, field := range fields {
		fieldName, exported := fieldName(field)
		if exported {
			fieldNames = append(fieldNames, fieldName)
		}
	}

	return fieldNames
}

// SliceOfFieldsWithVals generates slice of translated (cql tag) field names with field values
func SliceOfFieldsWithVals(val interface{} /*, opts Options*/) (fieldNames []string, vals []interface{}) {
	fields, vals := structs.ListExportedFieldsWithVals(val)

	fieldNames = []string{}
	for _, field := range fields {
		fieldName, exported := fieldName(field)
		if exported {
			fieldNames = append(fieldNames, fieldName)
		}
	}

	return fieldNames, vals
}

// updateSetExpToString generates UPDATE + SET part of SQL statement
// for fields of an entity
func updateSetExpToString(cfName string, val interface{} /*, opts Options*/) (
	statement string, fields []string, err error) {

	fields = sliceOfFieldNames(val)

	statement = updateStatement(cfName, fields)
	return statement, fields, nil
}

// UPDATE keyspace.Movies SET col1 = val1, col2 = val2
func updateStatement(cfName string, fields []string /*, opts Options*/) (statement string) {
	buf := new(bytes.Buffer)
	buf.WriteString(fmt.Sprintf("UPDATE %s ", cfName))

	/*
		// Apply options
		if opts.TTL != 0 {
			buf.WriteString("USING TTL ")
			buf.WriteString(strconv.FormatFloat(opts.TTL.Seconds(), 'f', 0, 64))
			buf.WriteRune(' ')
		}*/

	buf.WriteString("SET ")
	first := true
	for _, fieldName := range fields {
		if !first {
			buf.WriteString(", ")
		} else {
			first = false
		}
		buf.WriteString(fieldName)
		buf.WriteString(` = ?`)
	}

	return buf.String()
}

type findEntityVisitor struct {
	entity interface{}
}

// VisitPrefixedExp checks for "FROM" expression to find out the entity
func (visitor *findEntityVisitor) VisitPrefixedExp(exp *sql.PrefixedExp) {
	if exp.Prefix == "FROM" {
		if len(exp.Binding) == 1 && r.Indirect(r.ValueOf(exp.Binding[0])).Kind() == r.Struct {
			visitor.entity = exp.Binding[0]
		}
	} else if exp.AfterPrefix != nil {
		exp.AfterPrefix.Accept(visitor)
	}
}

// VisitFieldExpression just propagates to AfterFieldExpression
func (visitor *findEntityVisitor) VisitFieldExpression(exp *sql.FieldExpression) {
	if exp.AfterField != nil {
		exp.AfterField.Accept(visitor)
	}
}
