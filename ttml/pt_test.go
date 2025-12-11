package ttml

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
)

func TestPT(t *testing.T) {
	// 读取Bejeweled.ttml
	file, err := os.Open("../Bejeweled.ttml")
	if err != nil {
		t.Error(err)
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		t.Error(err)
	}
	t.Log(string(data))
	tt, err := ParseTTML(string(data))
	if err != nil {
		t.Error(err)
	}
	// 转换为json
	jsonData, err := json.Marshal(tt)
	if err != nil {
		t.Error(err)
	}
	// 保存
	err = ioutil.WriteFile("../Bejeweled.json", jsonData, 0644)
}
