package main

import (
	"encoding/json"
	"fmt"
)

type kineticaResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	DataType string `json:"data_type"`
	Data     string `json:"data"`
	DataStr  string `json:"data_str"`
}

func parseBodyAsKineticaResponse(body []byte) (kineticaResponse, error) {
	var kineticaResponse kineticaResponse
	err := json.Unmarshal(body, &kineticaResponse)
	if err != nil {
		return kineticaResponse, err
	}
	return kineticaResponse, nil
}

type executeSqlResponse struct {
	CountAffected         int32             `json:"count_affected"`
	ResponseSchemaStr     string            `json:"response_schema_str"`
	BinaryEncodedResponse string            `json:"binary_encoded_response"`
	JsonEncodedResponse   string            `json:"json_encoded_response"`
	TotalNumberOfRecords  int32             `json:"total_number_of_records"`
	HasMoreRecords        bool              `json:"has_more_records"`
	PagingTable           string            `json:"paging_table"`
	Info                  map[string]string `json:"info"`
}

func parseExecuteSqlResponse(data string) (executeSqlResponse, error) {
	var executeSqlResponse executeSqlResponse
	err := json.Unmarshal([]byte(data), &executeSqlResponse)
	if err != nil {
		return executeSqlResponse, err
	}
	return executeSqlResponse, nil
}

func parseJsonEncodedResponseAsListOfMaps(data string) ([]map[string]interface{}, error) {
	var jsonData map[string][]interface{}
	err := json.Unmarshal([]byte(data), &jsonData)
	if err != nil {
		return nil, err
	}
	var results []map[string]interface{}
	for r := 0; r < len(jsonData["column_1"]); r++ {
		o := make(map[string]interface{})
		for c := 0; c < len(jsonData["column_headers"]); c++ {
			o[jsonData["column_headers"][c].(string)] = jsonData["column_"+fmt.Sprint(c+1)][r]
		}
		results = append(results, o)
	}
	return results, nil
}
