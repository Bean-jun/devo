package main

import "testing"

// ─── 主题切换测试 ───

func TestToggleTheme(t *testing.T) {
	currentTheme = Dark
	if currentTheme.Name != "dark" {
		t.Error("初始主题应为 dark")
	}

	toggleTheme()
	if currentTheme.Name != "light" {
		t.Error("切换后主题应为 light")
	}

	toggleTheme()
	if currentTheme.Name != "dark" {
		t.Error("再次切换后主题应为 dark")
	}
}
