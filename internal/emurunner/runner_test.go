package emurunner

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// fakeSerializer 实现 stateSerializer，避免依赖真实 libretro（cgo）
type fakeSerializer struct {
	data       []byte
	serErr     error
	unserErr   error
	unserInput []byte
}

func (f *fakeSerializer) Serialize() ([]byte, error) {
	if f.serErr != nil {
		return nil, f.serErr
	}
	return f.data, nil
}

func (f *fakeSerializer) Unserialize(data []byte) error {
	f.unserInput = data
	return f.unserErr
}

func TestSaveStateTo(t *testing.T) {
	dir := t.TempDir()
	payload := []byte("nes-save-state-bytes")
	fs := &fakeSerializer{data: payload}

	if err := saveStateTo(fs, dir); err != nil {
		t.Fatalf("saveStateTo error: %v", err)
	}

	// state.dat 内容正确
	got, err := os.ReadFile(filepath.Join(dir, saveStateFile))
	if err != nil {
		t.Fatalf("read state.dat: %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("state.dat = %q, want %q", got, payload)
	}

	// state.done 标志存在
	if _, err := os.Stat(filepath.Join(dir, saveDoneFile)); err != nil {
		t.Errorf("state.done not created: %v", err)
	}

	// 无残留 .tmp
	if _, err := os.Stat(filepath.Join(dir, saveStateFile+".tmp")); !os.IsNotExist(err) {
		t.Error("残留 state.dat.tmp 未清理")
	}
}

func TestSaveStateTo_SerializeError(t *testing.T) {
	dir := t.TempDir()
	fs := &fakeSerializer{serErr: errors.New("boom")}

	if err := saveStateTo(fs, dir); err == nil {
		t.Fatal("期望序列化失败返回错误")
	}
	// 失败时不应写出完成标志
	if _, err := os.Stat(filepath.Join(dir, saveDoneFile)); !os.IsNotExist(err) {
		t.Error("序列化失败不应写出 state.done")
	}
}

func TestLoadStateFrom(t *testing.T) {
	dir := t.TempDir()
	payload := []byte("load-me")
	if err := os.WriteFile(filepath.Join(dir, loadStateFile), payload, 0644); err != nil {
		t.Fatal(err)
	}
	fs := &fakeSerializer{}

	if err := loadStateFrom(fs, dir); err != nil {
		t.Fatalf("loadStateFrom error: %v", err)
	}
	if string(fs.unserInput) != string(payload) {
		t.Errorf("unserialize input = %q, want %q", fs.unserInput, payload)
	}
	if _, err := os.Stat(filepath.Join(dir, loadDoneFile)); err != nil {
		t.Errorf("load.done not created: %v", err)
	}
}

func TestLoadStateFrom_MissingFile(t *testing.T) {
	dir := t.TempDir()
	fs := &fakeSerializer{}
	if err := loadStateFrom(fs, dir); err == nil {
		t.Fatal("期望缺少 load.dat 时返回错误")
	}
}
