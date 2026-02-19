package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"
)

func main() {
	// 1. フォーム入力
	config, err := RunForm()
	if err != nil {
		log.Fatal(err)
	}

	// 2. プログラムパスをフルパスに解決
	config.Program, err = resolveProgram(config.Program)
	if err != nil {
		log.Fatal(err)
	}

	// 3. plist生成
	plist, err := GeneratePlist(config)
	if err != nil {
		log.Fatal(err)
	}

	// 4. プレビュー表示
	fmt.Println("\n--- 生成されるplist ---")
	fmt.Println(plist)
	fmt.Println("--- ここまで ---")

	// 5. 既存ファイルの上書き確認
	plistPath, err := PlistPath(config.Label)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stat(plistPath); err == nil {
		var overwrite bool
		msg := fmt.Sprintf("%s は既に存在します。既存サービスを停止して上書きしますか？", plistPath)
		overwriteConfirm := huh.NewConfirm().
			Title(msg).
			Value(&overwrite)
		if err := overwriteConfirm.Run(); err != nil {
			log.Fatal(err)
		}
		if !overwrite {
			fmt.Println("キャンセルしました")
			return
		}
	}

	// 6. ロード確認
	var doLoad bool
	confirm := huh.NewConfirm().
		Title("LaunchAgentとしてロードしますか？").
		Value(&doLoad)

	if err := confirm.Run(); err != nil {
		log.Fatal(err)
	}

	if !doLoad {
		fmt.Println("キャンセルしました")
		return
	}

	// 7. インストール
	if err := Install(config.Label, plist); err != nil {
		log.Fatal(err)
	}
}

// resolveProgram はコマンド文字列の先頭プログラム名をフルパスに解決する。
// 既にフルパス（/で始まる）場合はそのまま返す。
// "mytool --flag" のように引数付きの場合、先頭部分のみ解決する。
func resolveProgram(program string) (string, error) {
	parts := strings.Fields(program)
	if len(parts) == 0 {
		return program, nil
	}
	cmd := parts[0]
	if strings.HasPrefix(cmd, "/") {
		return program, nil
	}
	resolved, err := exec.LookPath(cmd)
	if err != nil {
		return "", fmt.Errorf("コマンド %q が見つかりません: %w", cmd, err)
	}
	parts[0] = resolved
	return strings.Join(parts, " "), nil
}
