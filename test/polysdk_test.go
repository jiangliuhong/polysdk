package test

import (
	"encoding/json"
	"fmt"
	"github.com/jiangliuhong/polysdk/polysdk"
	"testing"
)

const AppId = "2dpjq"
const ProviderAttrDef = "ProviderAttrDef"

func getAuth() polysdk.Auth {
	return polysdk.NewAuth("jaromejiang@yunify.com", "Jiang123.")
}
func getModelClient(appId string, modelCode string) polysdk.DataModelClient {
	auth := getAuth()
	return polysdk.NewQxDataModelClient(auth, appId, modelCode)
}
func Test1GetToken(t *testing.T) {
	auth := getAuth()
	token, err := auth.GetToken()
	if err != nil {
		panic(err)
	}
	println("token:\n" + *token)
}

func Test2CreateModel(t *testing.T) {
	client := getModelClient(AppId, ProviderAttrDef)
	data := map[string]interface{}{
		"entity": map[string]interface{}{
			"display_type":    "string",
			"key":             "sUlxRQ9c",
			"name":            "单元测试添加",
			"provider_def_id": "0BA4311F7354479CA28F8FD7B224CEF1",
			"value_type":      "text",
		},
	}
	create, entity, err := client.Create(data)
	if err != nil {
		t.Fatal(err)
		return
	}
	if create {
		marshal, err := json.Marshal(entity)
		if err != nil {
			t.Fatal(err)
			return
		}
		println("create result:\n" + string(marshal))
	} else {
		t.Fatal("create fail")
	}
}

func Test3GetModel(t *testing.T) {
	client := getModelClient(AppId, ProviderAttrDef)
	entity, err := client.Get("DA4C39E9CA534BBD91E04EB2C38056A2")
	if err != nil {
		t.Fatal(err)
		return
	}
	paramJsonBytes, _ := json.Marshal(entity)
	println(string(paramJsonBytes))
}

func TestSearchModel(t *testing.T) {
	client := getModelClient(AppId, ProviderAttrDef)
	param := polysdk.DataModelSearchParam{
		Page: 1,
		Size: 20,
	}
	list, total, err := client.Search(param)
	if err != nil {
		t.Fatal(err)
		return
	}
	listByte, _ := json.Marshal(list)
	fmt.Printf("total:%v,list:%v", string(rune(total)), string(listByte))
}
