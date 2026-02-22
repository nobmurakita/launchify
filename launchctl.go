package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// plistDir はplistファイルの配置先ディレクトリを返す
func plistDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("ホームディレクトリの取得に失敗: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents"), nil
}

// PlistPath はラベルに対応するplistファイルのパスを返す
func PlistPath(label string) (string, error) {
	dir, err := plistDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, label+".plist"), nil
}

// Install はplist XMLをファイルに書き出し、launchctl load を実行する。
// 既存ファイルがある場合はパス指定でアンロードしてから上書きする。
// 進捗メッセージはwに出力する。
// TODO: launchctl load/unload は macOS 10.10 以降非推奨。将来的に bootstrap/bootout への移行を検討。
func Install(label, plistXML string, w io.Writer) error {
	path, err := PlistPath(label)
	if err != nil {
		return err
	}

	// 既存ファイルがあれば、中のラベルに関係なくアンロード（エラーは無視）
	if _, err := os.Stat(path); err == nil {
		cmd := exec.Command("launchctl", "unload", path)
		_ = cmd.Run()
		fmt.Fprintln(w, "既存サービスをアンロードしました")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("ディレクトリの作成に失敗: %w", err)
	}

	if err := os.WriteFile(path, []byte(plistXML), 0644); err != nil {
		return fmt.Errorf("plistファイルの書き出しに失敗: %w", err)
	}

	cmd := exec.Command("launchctl", "load", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl loadに失敗: %s: %w", string(output), err)
	}

	fmt.Fprintf(w, "インストール完了: %s\n", path)
	return nil
}
