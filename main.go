package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func main() {
	config, plist, err := RunApp()
	if err != nil {
		log.Fatal(err)
	}
	if config == nil {
		fmt.Println("キャンセルしました")
		return
	}
	if err := Install(config.Label, plist); err != nil {
		log.Fatal(err)
	}
}

// RunApp はフォーム入力→プレビュー確認のループを管理する。
// プレビューで「戻る」を選ぶとフォームに戻り、入力値は保持される。
// インストールを選ぶとConfig・plistを返し、終了を選ぶとnilを返す。
func RunApp() (*Config, string, error) {
	c := &Config{RunAtLoad: true}
	s := &formState{}

	for {
		form := buildForm(c, s)
		if err := form.WithTheme(formTheme()).Run(); err != nil {
			return nil, "", err
		}
		applyFormValues(c, s)

		resolved, err := resolveProgram(c.Program)
		if err != nil {
			return nil, "", err
		}
		c.Program = resolved

		plist, err := GeneratePlist(c)
		if err != nil {
			return nil, "", err
		}

		plistPath, err := PlistPath(c.Label)
		if err != nil {
			return nil, "", err
		}

		switch RunPreview(plist, plistPath) {
		case PreviewInstall:
			return c, plist, nil
		case PreviewBack:
			continue
		default: // PreviewQuit
			return nil, "", nil
		}
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
