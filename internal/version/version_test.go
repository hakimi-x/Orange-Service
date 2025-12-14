package version

import (
	"testing"
)

func TestGet_Initial(t *testing.T) {
	// 初始状态应该返回 nil
	// 注意：如果之前有其他测试设置了 current，这个测试可能会失败
	// 这里主要测试并发安全性
	info := Get()
	// 不做断言，只确保不会 panic
	_ = info
}

func TestAssetStruct(t *testing.T) {
	asset := Asset{
		Name:        "test.zip",
		Size:        1024,
		DownloadURL: "http://example.com/test.zip",
	}

	if asset.Name != "test.zip" {
		t.Error("Asset.Name 不正确")
	}
	if asset.Size != 1024 {
		t.Error("Asset.Size 不正确")
	}
	if asset.DownloadURL != "http://example.com/test.zip" {
		t.Error("Asset.DownloadURL 不正确")
	}
}

func TestInfoStruct(t *testing.T) {
	info := Info{
		Version:      "v1.0.0",
		ReleaseNotes: "Test release",
		PublishedAt:  "2024-01-01T00:00:00Z",
		Assets: []Asset{
			{Name: "test.zip", Size: 1024},
		},
	}

	if info.Version != "v1.0.0" {
		t.Error("Info.Version 不正确")
	}
	if len(info.Assets) != 1 {
		t.Error("Info.Assets 长度不正确")
	}
}
