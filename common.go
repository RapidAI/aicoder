package main

var trayTranslations = map[string]map[string]string{
	"en": {
		"title":   "Claude Config Manager",
		"show":    "Show Main Window",
		"launch":  "Launch Claude Code",
		"quit":    "Quit Application",
		"models":  "Models",
		"actions": "Actions",
	},
	"zh-Hans": {
		"title":   "Claude 配置管理器",
		"show":    "显示主窗口",
		"launch":  "启动 Claude Code",
		"quit":    "退出程序",
		"models":  "模型选择",
		"actions": "操作",
	},
	"zh-Hant": {
		"title":   "Claude 配置管理器",
		"show":    "顯示主視窗",
		"launch":  "啟動 Claude Code",
		"quit":    "退出程式",
		"models":  "模型選擇",
		"actions": "操作",
	},
	"ko": {
		"title":   "Claude 구성 관리자",
		"show":    "메인 창 표시",
		"launch":  "Claude Code 시작",
		"quit":    "프로그램 종료",
		"models":  "모델",
		"actions": "작업",
	},
	"ja": {
		"title":   "Claude 設定マネージャー",
		"show":    "メインウィンドウを表示",
		"launch":  "Claude Code を起動",
		"quit":    "終了",
		"models":  "モデル",
		"actions": "操作",
	},
	"de": {
		"title":   "Claude Konfigurationsmanager",
		"show":    "Hauptfenster anzeigen",
		"launch":  "Claude Code starten",
		"quit":    "Beenden",
		"models":  "Modelle",
		"actions": "Aktionen",
	},
	"fr": {
		"title":   "Gestionnaire de configuration Claude",
		"show":    "Afficher la fenêtre principale",
		"launch":  "Lancer Claude Code",
		"quit":    "Quitter",
		"models":  "Modèles",
		"actions": "Actions",
	},
}

const RequiredNodeVersion = "22.14.0"