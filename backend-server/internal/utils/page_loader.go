package utils

import (
	"os"
	"path/filepath"
)

// LoadDefaultPage pages/default.txtからHTMLを読み込む
// ファイルが存在しない場合はエラーを返す
func LoadDefaultPage() ([]byte, error) {
	projectRoot := FindProjectRoot()
	defaultPagePath := filepath.Join(projectRoot, "pages", "default.txt")
	return os.ReadFile(defaultPagePath)
}

// FindProjectRoot プロジェクトルート（go.modがあるディレクトリ）を探す
func FindProjectRoot() string {
	// 現在の実行ファイルのディレクトリから開始
	execPath, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(execPath)
		// go.modを探す
		for {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// 実行ファイルのパスが取得できない場合は、カレントディレクトリから探す
	wd, err := os.Getwd()
	if err == nil {
		dir := wd
		for {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// 見つからない場合はカレントディレクトリを返す
	return "."
}
