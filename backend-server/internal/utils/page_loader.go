package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// LoadDefaultPage 指定されたパスからHTMLを読み込む
// pagePathが相対パスの場合はプロジェクトルートからの相対パスとして扱う
// 絶対パスの場合はそのまま使用する
// ファイルが存在しない場合はエラーを返す
func LoadDefaultPage(pagePath string) ([]byte, error) {
	var fullPath string
	if filepath.IsAbs(pagePath) {
		// 絶対パスの場合はそのまま使用
		fullPath = pagePath
	} else {
		// 相対パスの場合はプロジェクトルートからの相対パスとして扱う
		projectRoot := FindProjectRoot()
		fullPath = filepath.Join(projectRoot, pagePath)
	}
	return os.ReadFile(fullPath)
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

// GetPlaceholderContent URLからコンテンツタイプを判定して適切なプレースホルダーを返す
// defaultDir: デフォルトページとプレースホルダーファイルのディレクトリ
// ファイルが存在する場合はファイルから読み込み、存在しない場合はコードで生成する
func GetPlaceholderContent(url string, defaultDir string) ([]byte, string, error) {
	urlLower := strings.ToLower(url)
	projectRoot := FindProjectRoot()

	// ディレクトリパスを解決
	var dirPath string
	if filepath.IsAbs(defaultDir) {
		dirPath = defaultDir
	} else {
		dirPath = filepath.Join(projectRoot, defaultDir)
	}

	// CSSファイル
	if strings.HasSuffix(urlLower, ".css") || strings.Contains(urlLower, "/css/") {
		filePath := filepath.Join(dirPath, "placeholder.css")
		if data, err := os.ReadFile(filePath); err == nil {
			return data, "text/css; charset=utf-8", nil
		}
		return []byte("/* CSS will be loaded from cache */"), "text/css; charset=utf-8", nil
	}

	// JavaScriptファイル
	if strings.HasSuffix(urlLower, ".js") || strings.Contains(urlLower, "/js/") {
		filePath := filepath.Join(dirPath, "placeholder.js")
		if data, err := os.ReadFile(filePath); err == nil {
			return data, "application/javascript; charset=utf-8", nil
		}
		return []byte("// JavaScript will be loaded from cache"), "application/javascript; charset=utf-8", nil
	}

	// 画像ファイル（PNG、JPG、GIF、SVG、WebP、ICO）
	if strings.HasSuffix(urlLower, ".png") {
		filePath := filepath.Join(dirPath, "placeholder.png")
		if data, err := os.ReadFile(filePath); err == nil {
			return data, "image/png", nil
		}
		return []byte{}, "image/png", nil
	}
	if strings.HasSuffix(urlLower, ".jpg") || strings.HasSuffix(urlLower, ".jpeg") {
		filePath := filepath.Join(dirPath, "placeholder.jpg")
		if data, err := os.ReadFile(filePath); err == nil {
			return data, "image/jpeg", nil
		}
		return []byte{}, "image/jpeg", nil
	}
	if strings.HasSuffix(urlLower, ".gif") {
		filePath := filepath.Join(dirPath, "placeholder.gif")
		if data, err := os.ReadFile(filePath); err == nil {
			return data, "image/gif", nil
		}
		return []byte{}, "image/gif", nil
	}
	if strings.HasSuffix(urlLower, ".svg") {
		filePath := filepath.Join(dirPath, "placeholder.svg")
		if data, err := os.ReadFile(filePath); err == nil {
			return data, "image/svg+xml", nil
		}
		return []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="1" height="1"></svg>`), "image/svg+xml", nil
	}
	if strings.HasSuffix(urlLower, ".webp") {
		filePath := filepath.Join(dirPath, "placeholder.webp")
		if data, err := os.ReadFile(filePath); err == nil {
			return data, "image/webp", nil
		}
		return []byte{}, "image/webp", nil
	}
	if strings.HasSuffix(urlLower, ".ico") {
		filePath := filepath.Join(dirPath, "placeholder.ico")
		if data, err := os.ReadFile(filePath); err == nil {
			return data, "image/x-icon", nil
		}
		return []byte{}, "image/x-icon", nil
	}

	// フォントファイル（WOFF、WOFF2、TTF、OTF）
	if strings.HasSuffix(urlLower, ".woff") {
		filePath := filepath.Join(dirPath, "placeholder.woff")
		if data, err := os.ReadFile(filePath); err == nil {
			return data, "font/woff", nil
		}
		return []byte{}, "font/woff", nil
	}
	if strings.HasSuffix(urlLower, ".woff2") {
		filePath := filepath.Join(dirPath, "placeholder.woff2")
		if data, err := os.ReadFile(filePath); err == nil {
			return data, "font/woff2", nil
		}
		return []byte{}, "font/woff2", nil
	}
	if strings.HasSuffix(urlLower, ".ttf") {
		filePath := filepath.Join(dirPath, "placeholder.ttf")
		if data, err := os.ReadFile(filePath); err == nil {
			return data, "font/ttf", nil
		}
		return []byte{}, "font/ttf", nil
	}
	if strings.HasSuffix(urlLower, ".otf") {
		filePath := filepath.Join(dirPath, "placeholder.otf")
		if data, err := os.ReadFile(filePath); err == nil {
			return data, "font/otf", nil
		}
		return []byte{}, "font/otf", nil
	}

	// その他（HTMLなど）はnilを返して、呼び出し元でデフォルトページを読み込む
	return nil, "", nil
}
