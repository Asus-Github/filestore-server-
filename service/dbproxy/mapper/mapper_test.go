package mapper

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestFuncCallValidation(t *testing.T) {
	if _, err := FuncCall("/no/such/func"); err == nil {
		t.Error("unknown func should fail")
	}
	// 参数数量不匹配
	if _, err := FuncCall("/user/GetUserInfo"); err == nil {
		t.Error("wrong param count should fail")
	}
	// 参数类型不匹配时返回错误而不是panic
	if _, err := FuncCall("/user/GetUserInfo", json.Number("1")); err == nil {
		t.Error("wrong param type should fail")
	}
	if _, err := FuncCall("/ufile/QueryUserFileMetas", "u", "x", json.Number("0")); err == nil {
		t.Error("string for int param should fail")
	}
	if _, err := FuncCall("/ufile/QueryUserFileMetas", "u", json.Number("1.5"), json.Number("0")); err == nil {
		t.Error("float for int param should fail")
	}
}

func TestConvertParam(t *testing.T) {
	v, err := convertParam(json.Number("42"), reflect.TypeOf(int64(0)))
	if err != nil || v.Int() != 42 {
		t.Fatalf("convert int: %v %v", v, err)
	}
	if _, err := convertParam(json.Number("300"), reflect.TypeOf(int8(0))); err == nil {
		t.Error("overflow should fail")
	}
	s, err := convertParam("abc", reflect.TypeOf(""))
	if err != nil || s.String() != "abc" {
		t.Fatalf("convert string: %v %v", s, err)
	}
	if _, err := convertParam(nil, reflect.TypeOf("")); err == nil {
		t.Error("nil for string should fail")
	}
}
