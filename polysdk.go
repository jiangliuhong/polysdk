package polysdk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

const (
	AccessTokenHeaderName = "Access-Token"
	JsonContentType       = "application/json"
	QxpBaseServerPath     = "http://api.clouden.io"
	QxpApiLoginPath       = "/api/v1/warden/login"
	DataModelPathPattern  = "/api/v1/polyapi/request/system/app/%v/raw/inner/form/%v/%v_%v.r"
)

type LoginType string

const (
	Pwd = LoginType("pwd")
)

// Auth 认证信息
type Auth struct {
	Username  string
	Password  string
	LoginType LoginType
	Token     string
	Expiry    *time.Time
}

// TokenResponseBody token返回结果
type TokenResponseBody struct {
	Code int `json:"code"`
	Data struct {
		AccessToken string     `json:"access_token"`
		Expiry      *time.Time `json:"expiry"`
	} `json:"data"`
	Msg string `json:"msg"`
}

func bodyClose(Body io.ReadCloser) {
	err := Body.Close()
	if err != nil {
		panic(err)
	}
}

// IsExpiry 判断token是否过期
// true：过期
func (auth *Auth) IsExpiry() bool {
	if auth.Expiry == nil {
		return true
	}
	subDuration := auth.Expiry.Sub(time.Now())
	subSecond := subDuration * time.Second
	return subSecond < 600
}

func (auth *Auth) GetToken() (*string, error) {
	if !auth.IsExpiry() {
		return &auth.Token, nil
	}
	tokenReq := map[string]string{
		"username":   auth.Username,
		"password":   auth.Password,
		"login_type": string(auth.LoginType),
	}
	tokenReqJsonByte, _ := json.Marshal(tokenReq)
	resp, err := http.Post(QxpBaseServerPath+QxpApiLoginPath, JsonContentType, bytes.NewReader(tokenReqJsonByte))
	if err != nil {
		return nil, err
	}
	defer bodyClose(resp.Body)
	bodyByte, err := ioutil.ReadAll(resp.Body)
	var responseBody TokenResponseBody
	err = json.Unmarshal(bodyByte, &responseBody)
	if err != nil {
		return nil, err
	}
	if responseBody.Code != 0 {
		return nil, errors.New(responseBody.Msg)
	}
	auth.Token = responseBody.Data.AccessToken
	auth.Expiry = responseBody.Data.Expiry
	auth.IsExpiry()
	return &auth.Token, nil
}

// NewAuth 生成一个认证对象
func NewAuth(username string, password string) Auth {
	return Auth{Username: username, Password: password, LoginType: Pwd}
}

// DataModelSearchParam 数据模型多条查询参数
type DataModelSearchParam struct {
	Query map[string]interface{} `json:"query"`
	Page  int                    `json:"page"`
	Size  int                    `json:"size"`
	Sort  []string               `json:"sort"`
}

// DataModelClient 数据模型客户端
type DataModelClient interface {
	// Create 创建一条数据
	Create(data map[string]interface{}) (bool, map[string]interface{}, error)
	// BatchCreate 批量创建数据
	BatchCreate(dataList []map[string]interface{}) ([]map[string]interface{}, error)
	// Get 查询一条数据
	Get(id string) (map[string]interface{}, error)
	// Search 查询多条
	Search(searchParam DataModelSearchParam) ([]map[string]interface{}, int, error)
	// Delete 删除一条
	Delete(id string) (bool, int, error)
	// DeleteByQuery 根据条件查询
	DeleteByQuery(query map[string]interface{}) (bool, int, error)
	// Update 更新
	Update(data map[string]interface{}, query map[string]interface{}) (bool, int, error)
}

// QxDataModelClient 全象模型客户端实现
type QxDataModelClient struct {
	Auth      Auth   // 认证信息
	AppId     string // 应用id
	ModelCode string // 模型标识
}

// NewQxDataModelClient 返回全象数据模型客户端对象
func NewQxDataModelClient(auth Auth, appId string, modeCode string) DataModelClient {
	return &QxDataModelClient{Auth: auth, AppId: appId, ModelCode: modeCode}
}

// 构建动作请求API路径
func (client QxDataModelClient) buildApiPath(action string) string {
	apiPath := fmt.Sprintf(DataModelPathPattern, client.AppId, client.ModelCode, client.ModelCode, action)
	return apiPath
}

// 执行http请求，并返回结果
func (client QxDataModelClient) doHttpPost(url string, requestByte []byte, v interface{}) error {
	request, err := http.NewRequest("POST", url, bytes.NewReader(requestByte))
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", JsonContentType)
	token, err := client.Auth.GetToken()
	if err != nil {
		return err
	}
	request.Header.Add(AccessTokenHeaderName, *token)
	httpClient := http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(request)
	//resp, err := http.Post(url, JsonContentType, bytes.NewReader(requestByte))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return errors.New("http error:" + strconv.Itoa(resp.StatusCode))
	}
	defer bodyClose(resp.Body)
	bodyByte, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(bodyByte, v)
	if err != nil {
		return err
	}
	return nil
}

// ModelResBody 模型新增、查询单个时的返回结果
type ModelResBody struct {
	Code int `json:"code"`
	Data struct {
		Count  int                    `json:"count"`
		Entity map[string]interface{} `json:"entity"`
	} `json:"data"`
	Msg string `json:"msg"`
}

// ModelSearchResBody 模型批量查询返回结果
type ModelSearchResBody struct {
	Code int `json:"code"`
	Data struct {
		Total    int                      `json:"total"`
		Entities []map[string]interface{} `json:"entities"`
	} `json:"data"`
	Msg string `json:"msg"`
}

func (client QxDataModelClient) Get(id string) (map[string]interface{}, error) {
	if len(id) == 0 {
		return nil, errors.New("id不能为空")
	}
	requestBody := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"_id": id,
			},
		},
	}
	requestBodyJsonByte, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	apiPath := client.buildApiPath("get")
	var getBody ModelResBody
	err = client.doHttpPost(QxpBaseServerPath+apiPath, requestBodyJsonByte, &getBody)
	if err != nil {
		return nil, err
	}
	if getBody.Code != 0 {
		return nil, errors.New(getBody.Msg)
	}
	return getBody.Data.Entity, nil
}

func (client QxDataModelClient) Create(data map[string]interface{}) (bool, map[string]interface{}, error) {
	if data == nil || len(data) == 0 {
		return false, nil, errors.New("参数不能为空")
	}
	apiPath := client.buildApiPath("create")
	requestBody := map[string]interface{}{
		"entity": data,
	}
	requestBodyJsonByte, err := json.Marshal(requestBody)
	if err != nil {
		return false, nil, err
	}
	var createBody ModelResBody
	err = client.doHttpPost(QxpBaseServerPath+apiPath, requestBodyJsonByte, &createBody)
	if createBody.Code != 0 {
		return false, nil, errors.New(createBody.Msg)
	}
	return true, createBody.Data.Entity, nil
}

func (client QxDataModelClient) BatchCreate(dataList []map[string]interface{}) ([]map[string]interface{}, error) {
	var list []map[string]interface{}
	for _, data := range dataList {
		_, entity, err := client.Create(data)
		if err != nil {
			return nil, err
		}
		list = append(list, entity)
	}
	return list, nil
}

func (client QxDataModelClient) Search(searchParam DataModelSearchParam) ([]map[string]interface{}, int, error) {
	apiPath := client.buildApiPath("search")
	if searchParam.Page == 0 {
		searchParam.Page = 1
	}
	if searchParam.Size == 0 {
		searchParam.Size = 20
	}
	requestBody, err := json.Marshal(searchParam)
	if err != nil {
		return nil, 0, err
	}
	var searchBody ModelSearchResBody
	err = client.doHttpPost(QxpBaseServerPath+apiPath, requestBody, &searchBody)
	if err != nil {
		return nil, 0, err
	}
	if searchBody.Code != 0 {
		return nil, 0, errors.New(searchBody.Msg)
	}
	if searchBody.Data.Entities == nil {
		return []map[string]interface{}{}, 0, nil
	}
	return searchBody.Data.Entities, searchBody.Data.Total, nil
}
func (client QxDataModelClient) Delete(id string) (bool, int, error) {
	if len(id) == 0 {
		return false, 0, errors.New("id不能为空")
	}
	apiPath := client.buildApiPath("delete")
	requestBody := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"_id": id,
			},
		},
	}
	requestBodyJsonByte, err := json.Marshal(requestBody)
	if err != nil {
		return false, 0, err
	}
	var deleteBody ModelResBody
	err = client.doHttpPost(QxpBaseServerPath+apiPath, requestBodyJsonByte, &deleteBody)
	if err != nil {
		return false, 0, err
	}
	if deleteBody.Code != 0 {
		return false, 0, errors.New(deleteBody.Msg)
	}
	return true, deleteBody.Data.Count, nil
}

// DeleteByQuery 根据条件查询
func (client QxDataModelClient) DeleteByQuery(query map[string]interface{}) (bool, int, error) {
	if query == nil || len(query) == 0 {
		return false, 0, errors.New("查询条件不能为空")
	}
	apiPath := client.buildApiPath("delete")
	requestBody := map[string]interface{}{
		"query": query,
	}
	requestBodyJsonByte, err := json.Marshal(requestBody)
	if err != nil {
		return false, 0, err
	}
	var deleteBody ModelResBody
	err = client.doHttpPost(QxpBaseServerPath+apiPath, requestBodyJsonByte, &deleteBody)
	if err != nil {
		return false, 0, err
	}
	if deleteBody.Code != 0 {
		return false, 0, errors.New(deleteBody.Msg)
	}
	return true, deleteBody.Data.Count, nil
}

// Update 更新
func (client QxDataModelClient) Update(data map[string]interface{}, query map[string]interface{}) (bool, int, error) {
	if data == nil || len(data) == 0 {
		return false, 0, errors.New("data不能为空")
	}
	requestBody := map[string]interface{}{
		"entity": data,
	}
	requestBodyJsonByte, err := json.Marshal(requestBody)
	if err != nil {
		return false, 0, err
	}
	apiPath := client.buildApiPath("update")
	var updateBody ModelResBody
	err = client.doHttpPost(QxpBaseServerPath+apiPath, requestBodyJsonByte, &updateBody)
	if err != nil {
		return false, 0, err
	}
	if updateBody.Code != 0 {
		return false, 0, errors.New(updateBody.Msg)
	}
	return true, updateBody.Data.Count, nil
}
