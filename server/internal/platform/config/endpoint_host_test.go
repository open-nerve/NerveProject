package config

import "testing"

func TestS3EndpointBrowserHost(t *testing.T) {
	for _, host := range []string{"127.1", "0177.0.0.1", "2130706433", "0x7f000001", "store.123", "store.0xabc", "store.0x", "127.0.0.1."} {
		t.Run(host, func(t *testing.T) {
			if _, err := loadS3Config("NERVE_FILES__S3__ENDPOINT=http://" + host); err == nil {
				t.Fatal("browser-incompatible numeric host accepted")
			}
		})
	}
	for _, host := range []string{"127.0.0.1", "s3.example.com", "123.example.com", "storage.invalid.", "store.0xgg", "localhost"} {
		t.Run("allow "+host, func(t *testing.T) {
			if _, err := loadS3Config("NERVE_FILES__S3__ENDPOINT=http://" + host); err != nil {
				t.Fatal(err)
			}
		})
	}
}
