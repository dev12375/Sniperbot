package util

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"io"
)

func CompressStruct(data any) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write(jsonData)
	w.Close()

	encoded := base64.URLEncoding.EncodeToString(b.Bytes())
	return encoded, nil
}

func DecompressStruct(compressed string, v any) error {
	decoded, err := base64.URLEncoding.DecodeString(compressed)
	if err != nil {
		return err
	}

	r, err := zlib.NewReader(bytes.NewReader(decoded))
	if err != nil {
		return err
	}
	defer r.Close()

	var b bytes.Buffer
	_, err = io.Copy(&b, r)
	if err != nil {
		return err
	}

	return json.Unmarshal(b.Bytes(), v)
}
