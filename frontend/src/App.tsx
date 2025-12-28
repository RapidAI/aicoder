import {useEffect, useState, useRef} from 'react';
import './App.css';
import {buildNumber} from './version';
import appIcon from './assets/images/appicon.png';
import {LoadConfig, SaveConfig, CheckEnvironment, ResizeWindow, LaunchClaude, SelectProjectDir, SetLanguage, GetUserHomeDir, CheckUpdate, RecoverCC, ShowMessage} from "../wailsjs/go/main/App";
import {WindowHide, EventsOn, EventsOff, BrowserOpenURL, ClipboardGetText, Quit} from "../wailsjs/runtime";
import {main} from "../wailsjs/go/models";

const subscriptionUrls: {[key: string]: string} = {
    "glm": "https://bigmodel.cn/glm-coding",
    "kimi": "https://www.kimi.com/membership/pricing?from=upgrade_plan&track_id=1d2446f5-f45f-4ae5-961e-c0afe936a115",
    "doubao": "https://www.volcengine.com/activity/codingplan",
    "minimax": "https://platform.minimaxi.com/user-center/payment/coding-plan"
};

const APP_VERSION = "1.3.2.1";

const translations: any = {
    "en": {
        "title": "Claude Code Easy Suite",
        "about": "About",
        "manual": "Manual",
        "cs146s": "Online Course",
        "recoverCC": "Recover CC",
        "hide": "Hide",
        "launch": "LAUNCH",
        "projectDir": "Project Directory",
        "change": "Change",
        "yoloMode": "Yolo Mode",
        "dangerouslySkip": "(Dangerously Skip Permissions)",
        "launchBtn": "Launch Claude Code",
        "activeModel": "ACTIVE MODEL",
        "modelSettings": "MODEL SETTINGS",
        "modelName": "Model Name",
        "apiKey": "API Key",
        "getKey": "Get API Key",
        "enterKey": "Enter API Key",
        "apiEndpoint": "API Endpoint",
        "saveChanges": "Save & Close",
        "saving": "Saving...",
        "saved": "Saved successfully!",
        "recovering": "Recovering...",
        "recoverSuccess": "Recovery successful!",
        "recoverSuccessAlert": "Claude Code has been reset. Please DO NOT click 'Launch Claude Code' here. Instead, open your terminal manually and run 'claude' to complete the native setup.",
        "confirmRecover": "Are you sure you want to recover Claude Code to its initial state? This will clear all configurations.",
        "recoverTitle": "Recover Claude Code",
        "recoverWarning": "Warning: This will permanently delete your Claude Code configurations and authentication tokens. This action cannot be undone.",
        "startRecover": "Start Recovery",
        "close": "Close",
        "manageProjects": "Manage Projects",
        "projectManagement": "Project Management",
        "projectName": "Project Name",
        "delete": "Delete",
        "addNewProject": "+ Add New Project",
        "projectDirError": "Please set a valid Project Directory!",
        "initializing": "Initializing...",
        "loadingConfig": "Loading config...",
        "syncing": "Syncing to Claude Code...",
        "switched": "Model switched & synced!",
        "projectSwitched": "Project switched!",
        "dirUpdated": "Directory updated!",
        "langName": "English",
        "custom": "Custom",
        "checkUpdate": "Check Update",
        "noUpdate": "No updates available",
        "updateAvailable": "Update available: ",
        "foundNewVersion": "Found new version",
        "downloadNow": "Download Now",
        "paste": "Paste",
        "bugReport": "Bug Report or Suggestion"
    },
    "zh-Hans": {
        "title": "Claude Code Easy Suite",
        "about": "关于",
        "manual": "使用说明",
        "cs146s": "在线课程",
        "recoverCC": "恢复CC",
        "hide": "隐藏",
        "launch": "启动",
        "projectDir": "项目目录",
        "change": "更改",
        "yoloMode": "Yolo 模式",
        "dangerouslySkip": "(危险：跳过权限检查)",
        "launchBtn": "启动 Claude Code",
        "activeModel": "模型选择",
        "modelSettings": "模型设置",
        "modelName": "模型名称",
        "apiKey": "API 密钥",
        "getKey": "获取API密钥",
        "enterKey": "输入 API Key",
        "apiEndpoint": "API 端点",
        "saveChanges": "保存并关闭",
        "saving": "保存中...",
        "saved": "保存成功！",
        "recovering": "正在恢复...",
        "recoverSuccess": "恢复成功！",
        "recoverSuccessAlert": "Claude Code 已重置。请注意：不要点击本程序的“启动 Claude Code”按钮。请自行手动打开终端窗口并运行 'claude' 命令以恢复原厂设置。",
        "confirmRecover": "确定要将 Claude Code 恢复到初始状态吗？这将清除所有配置。",
        "recoverTitle": "恢复 Claude Code",
        "recoverWarning": "警告：这将永久删除您的 Claude Code 配置和认证令牌。此操作无法撤销。",
        "startRecover": "开始恢复",
        "close": "关闭",
        "manageProjects": "项目管理",
        "projectManagement": "项目管理",
        "projectName": "项目名称",
        "delete": "删除",
        "addNewProject": "+ 添加新项目",
        "projectDirError": "请设置有效的项目目录！",
        "initializing": "初始化中...",
        "loadingConfig": "加载配置中...",
        "syncing": "正在同步到 Claude Code...",
        "switched": "模型已切换并同步！",
        "projectSwitched": "项目已切换！",
        "dirUpdated": "目录已更新！",
        "langName": "简体中文",
        "custom": "自定义",
        "checkUpdate": "检查更新",
        "noUpdate": "无可用更新",
        "updateAvailable": "发现新版本: ",
        "foundNewVersion": "发现新版本",
        "downloadNow": "立即下载",
        "paste": "粘贴",
        "bugReport": "Bug 报告或建议"
    },
    "zh-Hant": {
        "title": "Claude Code Easy Suite",
        "about": "關於",
        "manual": "使用說明",
        "cs146s": "線上課程",
        "recoverCC": "恢復CC",
        "hide": "隱藏",
        "launch": "啟動",
        "projectDir": "專案目錄",
        "change": "變更",
        "yoloMode": "Yolo 模式",
        "dangerouslySkip": "(危險：跳過權限檢查)",
        "launchBtn": "啟動 Claude Code",
        "activeModel": "模型選擇",
        "modelSettings": "模型設定",
        "modelName": "模型名稱",
        "apiKey": "API 金鑰",
        "getKey": "獲取API密鑰",
        "enterKey": "輸入 API Key",
        "apiEndpoint": "API 端點",
        "saveChanges": "儲存並關閉",
        "saving": "儲存中...",
        "saved": "儲存成功！",
        "recovering": "正在恢復...",
        "recoverSuccess": "恢復成功！",
        "recoverSuccessAlert": "Claude Code 已重置。請注意：不要點擊本程序的“啟動 Claude Code”按鈕。請自行手動打開終端窗口並運行 'claude' 命令以恢復原廠設置。",
        "confirmRecover": "確定要將 Claude Code 恢復到初始狀態嗎？這將清除所有配置。",
        "recoverTitle": "恢復 Claude Code",
        "recoverWarning": "警告：這將永久刪除您的 Claude Code 配置和認證令牌。此操作無法撤銷。",
        "startRecover": "開始恢復",
        "close": "關閉",
        "manageProjects": "專案管理",
        "projectManagement": "專案管理",
        "projectName": "專案名稱",
        "delete": "刪除",
        "addNewProject": "+ 新增專案",
        "projectDirError": "請設置有效的專案目錄！",
        "initializing": "初始化中...",
        "loadingConfig": "載入設定中...",
        "syncing": "正在同步到 Claude Code...",
        "switched": "模型已切換並同步！",
        "langName": "繁體中文",
        "custom": "自定義",
        "checkUpdate": "檢查更新",
        "noUpdate": "無可用更新",
        "updateAvailable": "發現新版本: ",
        "foundNewVersion": "發現新版本",
        "downloadNow": "立即下載",
        "paste": "貼上"
    },
    "ko": {
        "title": "Claude Code Easy Suite",
        "about": "정보",
        "manual": "매뉴얼",
        "cs146s": "온라인 강의",
        "recoverCC": "CC 초기화",
        "hide": "숨기기",
        "launch": "시작",
        "projectDir": "프로젝트 디렉토리",
        "change": "변경",
        "yoloMode": "Yolo 모드",
        "dangerouslySkip": "(위험: 권한 확인 건너뛰기)",
        "launchBtn": "Claude Code 시작",
        "activeModel": "모델 선택",
        "modelSettings": "모델 설정",
        "modelName": "모델 이름",
        "apiKey": "API 키",
        "getKey": "API 키 발급",
        "enterKey": "API 키 입력",
        "apiEndpoint": "API 엔드포인트",
        "saveChanges": "저장 및 닫기",
        "saving": "저장 중...",
        "saved": "저장 성공!",
        "recovering": "복구 중...",
        "recoverSuccess": "복구 성공!",
        "recoverSuccessAlert": "Claude Code가 초기화되었습니다.",
        "confirmRecover": "Claude Code를 초기 상태로 복구하시겠습니까? 모든 설정이 지워집니다.",
        "recoverTitle": "Claude Code 초기화",
        "recoverWarning": "경고: Claude Code 설정 및 인증 토큰이 영구적으로 삭제됩니다. 이 작업은 취소할 수 없습니다.",
        "startRecover": "초기화 시작",
        "close": "닫기",
        "manageProjects": "프로젝트 관리",
        "projectManagement": "프로젝트 관리",
        "projectName": "프로젝트 이름",
        "delete": "삭제",
        "addNewProject": "+ 새 프로젝트 추가",
        "projectDirError": "유효한 프로젝트 디렉토리를 설정해주세요!",
        "initializing": "초기화 중...",
        "loadingConfig": "설정 불러오는 중...",
        "syncing": "Claude Code와 동기화 중...",
        "switched": "모델 전환 및 동기화 완료!",
        "langName": "한국어",
        "custom": "사용자 정의",
        "paste": "붙여넣기"
    },
    "ja": {
        "title": "Claude Code Easy Suite",
        "about": "バージョン情報",
        "manual": "マニュアル",
        "cs146s": "オンライン講座",
        "recoverCC": "CCを復元",
        "hide": "隠す",
        "launch": "起動",
        "projectDir": "プロジェクト・ディレクトリ",
        "change": "変更",
        "yoloMode": "Yolo モード",
        "dangerouslySkip": "(危険：権限チェックをスキップ)",
        "launchBtn": "Claude Code を起動",
        "activeModel": "モデル選択",
        "modelSettings": "モデル設定",
        "modelName": "モデル名",
        "apiKey": "API キー",
        "getKey": "API キーを取得",
        "enterKey": "API キーを入力",
        "apiEndpoint": "API エンドポイント",
        "saveChanges": "保存して閉じる",
        "saving": "保存中...",
        "saved": "保存しました！",
        "recovering": "復元中...",
        "recoverSuccess": "復元成功！",
        "recoverSuccessAlert": "Claude Code はリセットされました。",
        "confirmRecover": "Claude Code を初期状態に復元しますか？すべての設定が消去されます。",
        "recoverTitle": "Claude Code の復元",
        "recoverWarning": "警告：Claude Code の設定と認証トークンが完全に削除されます。この操作は取り消せません。",
        "startRecover": "復元を開始",
        "close": "閉じる",
        "manageProjects": "プロジェクト管理",
        "projectManagement": "プロジェクト管理",
        "projectName": "プロジェクト名",
        "delete": "削除",
        "addNewProject": "+ 新規プロジェクト追加",
        "projectDirError": "有効なプロジェクトディレクトリを設定してください！",
        "initializing": "初期化中...",
        "loadingConfig": "設定を読み込み中...",
        "syncing": "Claude Code に同期中...",
        "switched": "モデルの切り替えと同期が完了しました！",
        "langName": "日本語",
        "custom": "カスタム",
        "paste": "貼り付け"
    },
    "de": {
        "title": "Claude Code Easy Suite",
        "about": "Über",
        "manual": "Handbuch",
        "cs146s": "Online-Kurs",
        "recoverCC": "CC wiederherstellen",
        "hide": "Verbergen",
        "launch": "Starten",
        "projectDir": "Projektverzeichnis",
        "change": "Ändern",
        "yoloMode": "Yolo-Modus",
        "dangerouslySkip": "(Gefahr: Berechtigungen überspringen)",
        "launchBtn": "Claude Code starten",
        "activeModel": "Aktives Modell",
        "modelSettings": "Modell-Einstellungen",
        "modelName": "Modellname",
        "apiKey": "API-Schlüssel",
        "getKey": "API-Schlüssel erhalten",
        "enterKey": "API-Schlüssel eingeben",
        "apiEndpoint": "API-Endpunkt",
        "saveChanges": "Speichern & Schließen",
        "saving": "Speichern...",
        "saved": "Erfolgreich gespeichert!",
        "recovering": "Wiederherstellen...",
        "recoverSuccess": "Wiederherstellung erfolgreich!",
        "recoverSuccessAlert": "Claude Code wurde zurückgesetzt.",
        "confirmRecover": "Möchten Sie Claude Code wirklich auf den Ausgangszustand zurücksetzen? Alle Konfigurationen werden gelöscht.",
        "recoverTitle": "Claude Code wiederherstellen",
        "recoverWarning": "Warnung: Dies löscht Ihre Claude Code-Konfigurationen und Authentifizierungstoken dauerhaft. Diese Aktion kann nicht rückgängig gemacht werden.",
        "startRecover": "Wiederherstellung starten",
        "close": "Schließen",
        "manageProjects": "Projektverwaltung",
        "projectManagement": "Projektverwaltung",
        "projectName": "Projektname",
        "delete": "Löschen",
        "addNewProject": "+ Neues Projekt hinzufügen",
        "projectDirError": "Bitte gültiges Projektverzeichnis festlegen!",
        "initializing": "Initialisiere...",
        "loadingConfig": "Lade Konfiguration...",
        "syncing": "Synchronisiere mit Claude Code...",
        "switched": "Modell gewechselt & synchronisiert!",
        "langName": "Deutsch",
        "custom": "Benutzerdefiniert",
        "paste": "Einfügen"
    },
    "fr": {
        "title": "Claude Code Easy Suite",
        "about": "À propos",
        "manual": "Manuel",
        "cs146s": "Cours en ligne",
        "recoverCC": "Récupérer CC",
        "hide": "Masquer",
        "launch": "Lancer",
        "projectDir": "Répertoire du projet",
        "change": "Changer",
        "yoloMode": "Mode Yolo",
        "dangerouslySkip": "(Danger : Ignorer les permissions)",
        "launchBtn": "Lancer Claude Code",
        "activeModel": "Modèle actif",
        "modelSettings": "Paramètres du modèle",
        "modelName": "Nom du modèle",
        "apiKey": "Clé API",
        "getKey": "Obtenir une clé API",
        "enterKey": "Entrer la clé API",
        "apiEndpoint": "Point de terminaison API",
        "saveChanges": "Enregistrer et Fermer",
        "saving": "Enregistrement...",
        "saved": "Enregistré avec succès !",
        "recovering": "Récupération...",
        "recoverSuccess": "Récupération réussie !",
        "recoverSuccessAlert": "Claude Code a été réinitialisé.",
        "confirmRecover": "Êtes-vous sûr de vouloir récupérer Claude Code à son état initial ? Toutes les configurations seront effacées.",
        "recoverTitle": "Récupérer Claude Code",
        "recoverWarning": "Attention : Cela supprimera définitivement vos configurations et jetons d'authentification Claude Code. Cette action est irréversible.",
        "startRecover": "Démarrer la récupération",
        "close": "Fermer",
        "manageProjects": "Gestion de projet",
        "projectManagement": "Gestion de projet",
        "projectName": "Nom du projet",
        "delete": "Supprimer",
        "addNewProject": "+ Ajouter un nouveau projet",
        "projectDirError": "Veuillez définir un répertoire de projet valide !",
        "initializing": "Initialisation...",
        "loadingConfig": "Chargement de la configuration...",
        "syncing": "Synchronisation avec Claude Code...",
        "switched": "Modèle changé et synchronisé !",
        "langName": "Français",
        "custom": "Personnalisé",
        "paste": "Coller"
    }
};

function App() {
    const [config, setConfig] = useState<main.AppConfig | null>(null);
    const [status, setStatus] = useState("");
    const [activeTab, setActiveTab] = useState(0);
    const [isLoading, setIsLoading] = useState(true);
    const [envLogs, setEnvLogs] = useState<string[]>(["Initializing..."]);
    const [showLogs, setShowLogs] = useState(false);
    const [yoloMode, setYoloMode] = useState(false);
    const [showAbout, setShowAbout] = useState(false);
    const [showModelSettings, setShowModelSettings] = useState(false);
    const [showProjectManager, setShowProjectManager] = useState(false);
    const [showUpdateModal, setShowUpdateModal] = useState(false);
    const [updateResult, setUpdateResult] = useState<any>(null);
    const [projectOffset, setProjectOffset] = useState(0);
    const [tempProjects, setTempProjects] = useState<any[]>([]); // Local state for project manager
    const [managerStatus, setManagerStatus] = useState("");
    const [lang, setLang] = useState("en");

    // Recover Modal State
    const [showRecoverModal, setShowRecoverModal] = useState(false);
    const [recoverLogs, setRecoverLogs] = useState<string[]>([]);
    const [recoverStatus, setRecoverStatus] = useState<"idle" | "recovering" | "success" | "error">("idle");
    const recoverLogRef = useRef<HTMLDivElement>(null);

    const logEndRef = useRef<HTMLTextAreaElement>(null);

    useEffect(() => {
        if (logEndRef.current) {
            logEndRef.current.scrollTop = logEndRef.current.scrollHeight;
        }
    }, [envLogs]);

    useEffect(() => {
        if (recoverLogRef.current) {
            recoverLogRef.current.scrollTop = recoverLogRef.current.scrollHeight;
        }
    }, [recoverLogs]);

    useEffect(() => {
        // Language detection
        const userLang = navigator.language;
        let initialLang = "en";
        if (userLang.startsWith("zh-TW") || userLang.startsWith("zh-HK")) {
            initialLang = "zh-Hant";
        } else if (userLang.startsWith("zh")) {
            initialLang = "zh-Hans";
        } else if (userLang.startsWith("ko")) {
            initialLang = "ko";
        } else if (userLang.startsWith("ja")) {
            initialLang = "ja";
        } else if (userLang.startsWith("de")) {
            initialLang = "de";
        } else if (userLang.startsWith("fr")) {
            initialLang = "fr";
        }
        setLang(initialLang);
        SetLanguage(initialLang);

        // Environment Check Logic
        const logHandler = (msg: string) => {
            setEnvLogs(prev => [...prev, msg]);
            if (msg.toLowerCase().includes("failed") || msg.toLowerCase().includes("error")) {
                setShowLogs(true);
            }
        };
        const doneHandler = () => {
            ResizeWindow(760, 520);
            setIsLoading(false);
        };

        EventsOn("env-log", logHandler);
        EventsOn("env-check-done", doneHandler);

        CheckEnvironment(); // Start checks

        // Config Logic
        LoadConfig().then((cfg) => {
            setConfig(cfg);
            if (cfg && cfg.models) {
                const idx = cfg.models.findIndex(m => m.model_name === cfg.current_model);
                if (idx !== -1) setActiveTab(idx);

                // Check if any model has an API key configured
                const hasAnyApiKey = cfg.models.some(m => m.api_key && m.api_key.trim() !== "");
                if (!hasAnyApiKey) {
                    setShowModelSettings(true);
                }
            }
        }).catch(err => {
            setStatus("Error loading config: " + err);
        });

        // Listen for external config changes (e.g. from Tray)
        // Only update the config state (Active Model UI), do NOT switch the editing Tab.
        const handleConfigChange = (cfg: main.AppConfig) => {
            setConfig(cfg);
        };
        EventsOn("config-changed", handleConfigChange);

        return () => {
            EventsOff("config-changed");
            EventsOff("env-log");
            EventsOff("env-check-done");
        };
    }, []);

    // Initialize temp projects when manager opens
    useEffect(() => {
        if (showProjectManager && config) {
            setTempProjects(JSON.parse(JSON.stringify(config.projects)));
            setManagerStatus("");
        }
    }, [showProjectManager, config]);

    const handleLangChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
        setLang(e.target.value);
        SetLanguage(e.target.value);
    };

    const t = (key: string) => {
        return translations[lang][key] || translations["en"][key] || key;
    };

    const handleApiKeyChange = (newKey: string) => {
        if (!config) return;
        const newModels = [...config.models];
        newModels[activeTab] = { ...newModels[activeTab], api_key: newKey };
        setConfig(new main.AppConfig({...config, models: newModels}));
    };

    const handleModelUrlChange = (newUrl: string) => {
        if (!config) return;
        const newModels = [...config.models];
        newModels[activeTab] = { ...newModels[activeTab], model_url: newUrl };
        setConfig(new main.AppConfig({...config, models: newModels}));
    };

    const handleModelNameChange = (newName: string) => {
        if (!config) return;
        const newModels = [...config.models];
        // If we rename the currently active model, we need to update current_model ID too
        const isRenamingActive = config.current_model === newModels[activeTab].model_name;
        newModels[activeTab] = { ...newModels[activeTab], model_name: newName };
        
        const newConfig = new main.AppConfig({
            ...config, 
            models: newModels,
            current_model: isRenamingActive ? newName : config.current_model
        });
        setConfig(newConfig);
    };

    const handleModelSwitch = (modelName: string) => {
        if (!config) return;
        
        // Find the model to verify if it has an API key
        const targetModel = config.models.find(m => m.model_name === modelName);
        if (!targetModel || !targetModel.api_key || targetModel.api_key.trim() === "") {
            setStatus("Please configure API Key first!");
            // Set active tab to this model so the user lands on the correct settings page
            const idx = config.models.findIndex(m => m.model_name === modelName);
            if (idx !== -1) setActiveTab(idx);
            
            setShowModelSettings(true);
            setTimeout(() => setStatus(""), 2000);
            return;
        }

        const newConfig = new main.AppConfig({...config, current_model: modelName});
        setConfig(newConfig);
        setStatus(t("syncing"));
        SaveConfig(newConfig).then(() => {
            setStatus(t("switched"));
            setTimeout(() => setStatus(""), 1500);
        }).catch(err => {
            setStatus("Error syncing: " + err);
        });
    };

    // Project Management Functions
    const getCurrentProject = () => {
        if (!config || !config.projects) return null;
        return config.projects.find((p: any) => p.id === config.current_project) || config.projects[0];
    };

    const handleProjectSwitch = (projectId: string) => {
        if (!config) return;
        const newConfig = new main.AppConfig({...config, current_project: projectId});
        setConfig(newConfig);
        setStatus(t("projectSwitched"));
        setTimeout(() => setStatus(""), 1500);
        SaveConfig(newConfig);
    };

    const handleSelectDir = () => {
        if (!config) return;
        SelectProjectDir().then((dir) => {
            if (dir && dir.length > 0) {
                const currentProj = getCurrentProject();
                if (!currentProj) return;

                const newProjects = config.projects.map((p: any) => 
                    p.id === currentProj.id ? { ...p, path: dir } : p
                );
                
                // Update deprecated project_dir for backward compat if needed, but primarily use projects list
                const newConfig = new main.AppConfig({...config, projects: newProjects, project_dir: dir});
                setConfig(newConfig);
                setStatus(t("dirUpdated"));
                setTimeout(() => setStatus(""), 1500);
                SaveConfig(newConfig);
            }
        });
    };

    const handleYoloChange = (checked: boolean) => {
        if (!config) return;
        const currentProj = getCurrentProject();
        if (!currentProj) return;

        const newProjects = config.projects.map((p: any) => 
            p.id === currentProj.id ? { ...p, yolo_mode: checked } : p
        );
        
        const newConfig = new main.AppConfig({...config, projects: newProjects});
        setConfig(newConfig);
        setStatus(t("saved"));
        setTimeout(() => setStatus(""), 1500);
        SaveConfig(newConfig);
    };

    // Temp Project Manager Handlers (Local State)
    const validateTempProjects = (projects: any[]) => {
        const names = projects.map(p => p.name.trim());
        if (names.some(n => n === "")) {
            setManagerStatus("Error: Project name cannot be empty.");
            return false;
        }
        const hasDuplicate = names.some((name, index) => names.indexOf(name) !== index);
        if (hasDuplicate) {
            setManagerStatus("Error: Duplicate project names are not allowed.");
            return false;
        }
        setManagerStatus("");
        return true;
    };

    const handleAddTempProject = async () => {
        let baseName = "Project";
        let newName = "";
        let i = 1;
        // Search for a truly unique name
        while (true) {
            newName = `${baseName} ${i}`;
            // eslint-disable-next-line
            if (!tempProjects.some((p: any) => p.name === newName)) break;
            i++;
        }

        const homeDir = await GetUserHomeDir();
        const newId = Math.random().toString(36).substr(2, 9);
        const newProject = {
            id: newId,
            name: newName,
            path: homeDir || "",
            yolo_mode: false
        };
        const newList = [...tempProjects, newProject];
        setTempProjects(newList);
        validateTempProjects(newList);
    };

    const handleDeleteTempProject = (id: string) => {
        if (tempProjects.length <= 1) return;
        const newList = tempProjects.filter((p: any) => p.id !== id);
        setTempProjects(newList);
        validateTempProjects(newList);
    };

    const handleRenameTempProject = (id: string, newName: string) => {
        const newList = tempProjects.map((p: any) => 
            p.id === id ? { ...p, name: newName } : p
        );
        setTempProjects(newList);
        validateTempProjects(newList);
    };

    const saveProjectManagerChanges = () => {
        if (!config) return;
        if (!validateTempProjects(tempProjects)) return;
        
        // Determine current project ID (keep if exists, else first available)
        let newCurrentId = config.current_project;
        if (!tempProjects.find(p => p.id === newCurrentId)) {
            newCurrentId = tempProjects.length > 0 ? tempProjects[0].id : "";
        }

        const newConfig = new main.AppConfig({
            ...config, 
            projects: tempProjects,
            current_project: newCurrentId
        });
        
        setConfig(newConfig);
        SaveConfig(newConfig);
        setShowProjectManager(false);
        
        // Adjust tabs offset if current selection is out of view, or just reset
        if (tempProjects.length <= 5) setProjectOffset(0);
    };

    const handleOpenSubscribe = (modelName: string) => {
        const url = subscriptionUrls[modelName.toLowerCase()];
        if (url) {
            BrowserOpenURL(url);
        }
    };

    const handleOpenManual = () => {
        const isChinese = lang === "zh-Hans" || lang === "zh-Hant";
        const url = isChinese 
            ? "https://github.com/RapidAI/cceasy/blob/main/UserManual_CN.md" 
            : "https://github.com/RapidAI/cceasy/blob/main/UserManual_EN.md";
        BrowserOpenURL(url);
    };

    const save = () => {
        if (!config) return;
        setStatus(t("saving"));
        SaveConfig(config).then(() => {
            setStatus(t("saved"));
            setTimeout(() => {
                setStatus("");
                setShowModelSettings(false);
            }, 1000);
        }).catch(err => {
            setStatus("Error saving: " + err);
        });
    };

    const handleStartRecover = () => {
        setRecoverStatus("recovering");
        setRecoverLogs([]);
        
        // Setup listener
        EventsOn("recover-log", (msg: string) => {
            setRecoverLogs(prev => [...prev, msg]);
        });

        RecoverCC().then(() => {
            setRecoverStatus("success");
            setRecoverLogs(prev => [...prev, "DONE!"]);
            EventsOff("recover-log");
            ShowMessage(t("recoverTitle"), t("recoverSuccessAlert"));
        }).catch((err) => {
            setRecoverStatus("error");
            setRecoverLogs(prev => [...prev, "Error: " + err]);
            EventsOff("recover-log");
        });
    };

    if (isLoading) {
        return (
            <div style={{
                height: '100vh', 
                display: 'flex', 
                flexDirection: 'column', 
                justifyContent: 'center', 
                alignItems: 'center', 
                backgroundColor: '#fff',
                padding: '20px',
                textAlign: 'center',
                boxSizing: 'border-box'
            }}>
                <h2 style={{color: '#fb923c', marginBottom: '20px'}}>Claude Code Easy Suite</h2>
                <div style={{width: '100%', height: '4px', backgroundColor: '#ffedd5', borderRadius: '2px', overflow: 'hidden', marginBottom: '15px'}}>
                    <div style={{
                        width: '50%', 
                        height: '100%', 
                        backgroundColor: '#fb923c', 
                        borderRadius: '2px', 
                        animation: 'indeterminate 1.5s infinite linear'
                    }}></div>
                </div>
                
                {showLogs ? (
                    <textarea 
                        ref={logEndRef}
                        readOnly
                        value={envLogs.join('\n')}
                        style={{
                            width: '100%',
                            height: '240px',
                            padding: '10px',
                            fontSize: '0.85rem',
                            fontFamily: 'monospace',
                            color: '#4b5563',
                            backgroundColor: '#fffdfa',
                            border: '1px solid #ffedd5',
                            borderRadius: '8px',
                            resize: 'none',
                            outline: 'none',
                            marginBottom: '10px'
                        }}
                    />
                ) : (
                    <div style={{fontSize: '0.9rem', color: '#6b7280', marginBottom: '15px', height: '20px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap'}}>
                        {envLogs[envLogs.length - 1]}
                    </div>
                )}

                <div style={{display: 'flex', gap: '15px', alignItems: 'center'}}>
                    <button 
                        onClick={() => setShowLogs(!showLogs)}
                        style={{
                            background: 'none',
                            border: 'none',
                            color: '#fb923c',
                            fontSize: '0.8rem',
                            cursor: 'pointer',
                            textDecoration: 'underline'
                        }}
                    >
                        {showLogs ? (lang === 'zh-Hans' ? '隐藏详情' : 'Hide Details') : (lang === 'zh-Hans' ? '查看详情' : 'Show Details')}
                    </button>

                    {showLogs && (
                        <button 
                            onClick={Quit}
                            style={{
                                backgroundColor: '#ef4444',
                                color: 'white',
                                border: 'none',
                                borderRadius: '4px',
                                padding: '4px 12px',
                                fontSize: '0.8rem',
                                cursor: 'pointer'
                            }}
                        >
                            {lang === 'zh-Hans' ? '退出程序' : 'Quit Application'}
                        </button>
                    )}
                </div>
                
                <style>{`
                    @keyframes indeterminate {
                        0% { transform: translateX(-100%); }
                        100% { transform: translateX(200%); }
                    }
                `}</style>
            </div>
        );
    }

    if (!config) return <div className="main-content" style={{display:'flex', justifyContent:'center', alignItems:'center'}}>{t("loadingConfig")}</div>;

    const currentModelConfig = config.models[activeTab];
    const currentProject = getCurrentProject();
    const visibleProjects = config.projects ? config.projects.slice(projectOffset, projectOffset + 5) : [];

    return (
        <div id="App">
            {/* Drag Handle */}
            <div style={{
                height: '30px', 
                width: '100%', 
                position: 'absolute', 
                top: 0, 
                left: 0, 
                zIndex: 999, 
                '--wails-draggable': 'drag'
            } as any}></div>

            <div className="header">
                 <div style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center'}}>
                    <div style={{display: 'flex', alignItems: 'center', gap: '12px'}}>
                        <div style={{position: 'relative', display: 'flex', alignItems: 'center', justifyContent: 'center'}}>
                            <div style={{position: 'absolute', width: '32px', height: '32px', background: 'rgba(249, 115, 22, 0.4)', filter: 'blur(8px)', borderRadius: '50%'}}></div>
                            <img src={appIcon} alt="App Icon" style={{width: '28px', height: '28px', position: 'relative', borderRadius: '6px'}} />
                        </div>
                        <h2 style={{
                            margin: 0, 
                            fontSize: '1.25rem',
                            fontWeight: 'bold',
                            background: 'linear-gradient(135deg, #FB923C 0%, #F97316 50%, #EA580C 100%)', 
                            WebkitBackgroundClip: 'text', 
                            WebkitTextFillColor: 'transparent',
                            filter: 'drop-shadow(0 2px 4px rgba(249, 115, 22, 0.15))'
                        }}>{t("title")}</h2>
                    </div>
                    <div style={{display: 'flex', gap: '10px', alignItems: 'center', '--wails-draggable': 'no-drag', zIndex: 1000, position: 'relative'} as any}>
                        <select
                            value={lang}
                            onChange={handleLangChange}
                            className="btn-link"
                            style={{
                                appearance: 'none', 
                                border: 'none', 
                                background: 'transparent', 
                                cursor: 'pointer',
                                outline: 'none'
                            }}
                        >
                            <option value="en">English</option>
                            <option value="zh-Hans">简体中文</option>
                            <option value="zh-Hant">繁體中文</option>
                            <option value="ko">한국어</option>
                            <option value="ja">日本語</option>
                            <option value="de">Deutsch</option>
                            <option value="fr">Français</option>
                        </select>
                        <div style={{
                            display: 'grid', 
                            gridTemplateColumns: 'repeat(3, auto)', 
                            gap: '8px 10px', 
                            alignItems: 'center',
                            justifyItems: 'end'
                        }}>
                            <button className="btn-link" onClick={() => setShowAbout(true)}>{t("about")}</button>
                            <button className="btn-link" onClick={handleOpenManual}>{t("manual")}</button>
                            <button onClick={WindowHide} className="btn-hide" style={{margin: 0, height: '24px', display: 'flex', alignItems: 'center', justifyContent: 'center'}}>
                                {t("hide")}
                            </button>
                            
                            <button 
                                className="btn-link" 
                                onClick={() => {
                                    setStatus(t("checkUpdate") + "...");
                                    CheckUpdate(APP_VERSION).then((result: any) => {
                                        setUpdateResult(result);
                                        setShowUpdateModal(true);
                                        setStatus("");
                                    }).catch((err: any) => {
                                        setStatus("Error: " + err);
                                    });
                                }}
                            >
                                {t("checkUpdate")}
                            </button>
                            <button 
                                className="btn-link" 
                                onClick={() => {
                                    setRecoverStatus("idle");
                                    setShowRecoverModal(true);
                                }}
                            >
                                {t("recoverCC")}
                            </button>
                            <button className="btn-link" onClick={() => BrowserOpenURL("https://github.com/BIT-ENGD/cs146s_cn")}>{t("cs146s")}</button>
                        </div>
                    </div>
                 </div>
            </div>

            <div className="main-content" style={{overflowY: currentModelConfig.is_custom ? 'auto' : 'hidden'}}>
                <div style={{
                    backgroundColor: '#fffbf5', 
                    margin: '0 10px 15px 10px', 
                    padding: '10px 10px 15px 10px', 
                    borderRadius: '12px',
                    border: '1px solid rgba(251, 146, 60, 0.1)'
                }}>
                    <div style={{position: 'relative', display: 'flex', justifyContent: 'center', alignItems: 'center', marginBottom: '5px'}}>
                        <h3 style={{fontSize: '1.1rem', color: '#fb923c', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: '5px', marginTop: '-5px', textAlign: 'center'}}>{t("activeModel")}</h3>
                        <button 
                            className="btn-link" 
                            onClick={() => setShowModelSettings(true)}
                            style={{
                                position: 'absolute', 
                                right: '10px', 
                                borderColor: '#fb923c', 
                                color: '#fb923c',
                                fontSize: '0.8rem'
                            }}
                        >
                            ⚙️ {t("modelSettings")}
                        </button>
                    </div>
                    <div className="model-switcher" style={{justifyContent: 'center', padding: '0 10px', marginBottom: 0}}>
                        {config.models.map((model) => (
                            <button
                                key={model.model_name}
                                className={`model-btn ${config.current_model === model.model_name ? 'selected' : ''}`}
                                onClick={() => handleModelSwitch(model.model_name)}
                                style={{
                                    textAlign: 'center',
                                    borderBottom: (model.api_key && model.api_key.trim() !== "") ? '3px solid #fb923c' : '1px solid var(--border-color)'
                                }}
                            >
                                {model.model_name}
                            </button>
                        ))}
                    </div>
                </div>

                <div style={{
                    backgroundColor: '#fffbf5',
                    margin: '0px 10px 0px 10px',
                    padding: '15px 15px 10px 15px',
                    borderRadius: '12px',
                    border: '1px solid rgba(251, 146, 60, 0.1)',
                    position: 'relative'
                }}>
                    <div style={{display: 'flex', justifyContent: 'center', alignItems: 'center', marginBottom: '15px', position: 'relative'}}>
                        <h3 style={{fontSize: '1.1rem', color: '#fb923c', textTransform: 'uppercase', letterSpacing: '0.05em', margin: 0, textAlign: 'center'}}>Vibe Coding</h3>
                        <button 
                            className="btn-link" 
                            onClick={() => setShowProjectManager(true)}
                            style={{
                                position: 'absolute', 
                                right: '0', 
                                borderColor: '#fb923c', 
                                color: '#fb923c',
                                fontSize: '0.8rem'
                            }}
                        >
                            📂 {t("manageProjects")}
                        </button>
                    </div>

                    <div className="tabs" style={{marginBottom: '15px', borderBottom: '1px solid var(--border-color)', justifyContent: 'flex-start', overflowX: 'auto'}}>
                        {projectOffset > 0 && (
                            <button 
                                className="tab-button" 
                                onClick={() => setProjectOffset(prev => Math.max(0, prev - 1))}
                                title="Previous Projects"
                                style={{padding: '5px 10px'}}
                            >
                                ◀
                            </button>
                        )}
                        {visibleProjects.map((proj: any) => (
                            <button
                                key={proj.id}
                                className={`tab-button ${config.current_project === proj.id ? 'active' : ''}`}
                                onClick={() => handleProjectSwitch(proj.id)}
                            >
                                {proj.name}
                            </button>
                        ))}
                        {config.projects && config.projects.length > projectOffset + 5 && (
                            <button 
                                className="tab-button" 
                                onClick={() => setProjectOffset(prev => (prev + 5 < config.projects.length ? prev + 1 : prev))}
                                title="Next Projects"
                                style={{padding: '5px 10px'}}
                            >
                                ▶
                            </button>
                        )}
                    </div>

                    {currentProject && (
                        <>
                            <div className="form-group" style={{display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '10px'}}>
                                <label className="form-label" style={{marginBottom: 0, whiteSpace: 'nowrap', textAlign: 'left', minWidth: '110px'}}>{t("projectDir")}:</label>
                                <div style={{display: 'flex', gap: '10px', flexGrow: 1}}>
                                    <input 
                                        type="text" 
                                        className="form-input"
                                        value={currentProject.path} 
                                        readOnly
                                        style={{backgroundColor: '#f9fafb', color: '#6b7280', flexGrow: 1, textAlign: 'left', padding: '10px'}}
                                    />
                                    <button className="btn-primary" style={{padding: '10px 15px', whiteSpace: 'nowrap'}} onClick={handleSelectDir}>{t("change")}</button>
                                </div>
                            </div>

                            <div style={{marginBottom: '10px'}}>
                                <label className="form-label" style={{display:'flex', alignItems:'center', cursor:'pointer'}}>
                                    <input 
                                        type="checkbox" 
                                        checked={currentProject.yolo_mode}
                                        onChange={(e) => handleYoloChange(e.target.checked)}
                                        style={{marginRight: '8px', transform: 'scale(1.2)'}}
                                    />
                                    <span style={{fontWeight: 600}}>{t("yoloMode")}</span> 
                                    <span style={{marginLeft:'8px', color:'#ef4444', fontSize:'0.85em'}}>{t("dangerouslySkip")}</span>
                                </label>
                            </div>

                            <button className="btn-launch" onClick={() => {
                                if (!currentProject.path || currentProject.path.trim() === "") {
                                    setStatus(t("projectDirError"));
                                    setTimeout(() => setStatus(""), 2000);
                                    return;
                                }
                                LaunchClaude(currentProject.yolo_mode, currentProject.path || "")
                            }}>
                                {t("launchBtn")}
                            </button>
                            <div className="status-message" style={{marginTop: '5px', minHeight: '20px'}}>
                                <span key={status} style={{color: (status.includes("Error") || status.includes("！") || status.includes("!") || status.includes("first")) ? '#ef4444' : '#10b981'}}>{status}</span>
                            </div>
                        </>
                    )}
                </div>
                
                                        </div>
                            
                                        {/* Model Settings Modal */}
                                    {showModelSettings && (
            
                                        <div className="modal-overlay" onClick={(e) => { if (e.target === e.currentTarget) setShowModelSettings(false); }}>
            
                                            <div className="modal-content" onClick={e => e.stopPropagation()} style={{width: '600px', textAlign: 'left'}}>
            
                                                <button className="modal-close" onClick={() => setShowModelSettings(false)}>&times;</button>
            
                                                
            
                                                <div style={{padding: '0 10px', display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '15px'}}>
                            <h3 style={{fontSize: '1.1rem', color: '#fb923c', textTransform: 'uppercase', letterSpacing: '0.05em', margin: 0}}>{t("modelSettings")}</h3>
                            <div style={{display: 'flex', alignItems: 'center'}}>
                                <div className="status-message" style={{margin: '0 15px 0 0', minHeight: 'auto'}}>
                                    <span key={status} style={{color: status.includes("Error") ? 'red' : 'green'}}>{status}</span>
                                </div>
                                <button className="btn-primary" style={{padding: '5px 15px', marginRight: '30px'}} onClick={save}>{t("saveChanges")}</button>
                            </div>
                        </div>

                        <div className="tabs" style={{padding: '0 10px'}}>
                            {config.models.map((model, index) => (
                                <button
                                    key={model.is_custom ? "custom-tab" : model.model_name}
                                    className={`tab-button ${activeTab === index ? 'active' : ''}`}
                                    onClick={() => setActiveTab(index)}
                                >
                                    {model.is_custom ? t("custom") || "Custom" : model.model_name}
                                </button>
                            ))}
                        </div>

                        <div style={{padding: '0 10px'}}>
                            {currentModelConfig.is_custom && (
                            <div className="form-group">
                                <label className="form-label">{t("modelName")}</label>
                                <input 
                                    type="text" 
                                    className="form-input"
                                    value={currentModelConfig.model_name} 
                                    onChange={(e) => handleModelNameChange(e.target.value)}
                                    placeholder="e.g. claude-3-5-sonnet-20241022"
                                />
                            </div>
                            )}

                            <div className="form-group">
                                <label className="form-label">{t("apiKey")}</label>
                                <div style={{display: 'flex', gap: '10px'}}>
                                    <input 
                                        type="password" 
                                        className="form-input"
                                        value={currentModelConfig.api_key} 
                                        onChange={(e) => handleApiKeyChange(e.target.value)}
                                        placeholder={`${t("enterKey")} (${currentModelConfig.model_name})`}
                                    />
                                    <button 
                                        className="btn-subscribe" 
                                        onClick={async () => {
                                            const text = await ClipboardGetText();
                                            if (text) handleApiKeyChange(text);
                                        }}
                                        title={t("paste")}
                                    >
                                        📋
                                    </button>
                                    {!currentModelConfig.is_custom && (
                                    <button 
                                        className="btn-subscribe" 
                                        onClick={() => handleOpenSubscribe(currentModelConfig.model_name)}
                                    >
                                        {t("getKey")}
                                    </button>
                                    )}
                                </div>
                            </div>

                            {currentModelConfig.is_custom && (
                            <div className="form-group">
                                <label className="form-label">{t("apiEndpoint")}</label>
                                <input 
                                    type="text" 
                                    className="form-input"
                                    value={currentModelConfig.model_url} 
                                    onChange={(e) => handleModelUrlChange(e.target.value)}
                                    placeholder="https://api.example.com/v1"
                                />
                            </div>
                            )}
                        </div>
                    </div>
                </div>
            )}

            {/* Recover CC Modal */}
            {showRecoverModal && (
                <div className="modal-overlay">
                    <div className="modal-content" onClick={e => e.stopPropagation()} style={{width: '500px', textAlign: 'left'}}>
                        {recoverStatus !== "recovering" && (
                            <button className="modal-close" onClick={() => setShowRecoverModal(false)}>&times;</button>
                        )}
                        <h3 style={{marginTop: 0, color: '#fb923c', marginBottom: '15px'}}>{t("recoverTitle")}</h3>
                        
                        {recoverStatus === "idle" && (
                            <div style={{marginBottom: '20px', color: '#ef4444', border: '1px solid #fca5a5', padding: '10px', borderRadius: '6px', backgroundColor: '#fef2f2'}}>
                                <p style={{margin: 0, fontWeight: 500}}>{t("recoverWarning")}</p>
                            </div>
                        )}

                        {recoverStatus !== "idle" && (
                            <div style={{
                                width: '100%',
                                height: '200px',
                                backgroundColor: '#1e1e1e',
                                color: '#e5e5e5',
                                padding: '10px',
                                borderRadius: '6px',
                                fontFamily: 'monospace',
                                fontSize: '0.85rem',
                                overflowY: 'auto',
                                marginBottom: '20px'
                            }}>
                                {recoverLogs.map((log, i) => (
                                    <div key={i} style={{marginBottom: '2px'}}>{log}</div>
                                ))}
                                <div ref={recoverLogRef} />
                            </div>
                        )}

                        <div style={{display: 'flex', justifyContent: 'flex-end', gap: '10px'}}>
                            {recoverStatus === "idle" ? (
                                <>
                                    <button 
                                        className="btn-primary" 
                                        style={{backgroundColor: '#6b7280'}}
                                        onClick={() => setShowRecoverModal(false)}
                                    >
                                        {t("close")}
                                    </button>
                                    <button 
                                        className="btn-primary" 
                                        style={{backgroundColor: '#ef4444'}}
                                        onClick={handleStartRecover}
                                    >
                                        {t("startRecover")}
                                    </button>
                                </>
                            ) : recoverStatus === "success" || recoverStatus === "error" ? (
                                <button 
                                    className="btn-primary" 
                                    onClick={() => setShowRecoverModal(false)}
                                >
                                    {t("close")}
                                </button>
                            ) : null}
                        </div>
                    </div>
                </div>
            )}

            {/* Project Manager Modal */}
            {showProjectManager && (
                <div className="modal-overlay">
                    <div className="modal-content" onClick={e => e.stopPropagation()} style={{width: '500px', textAlign: 'left'}}>
                        <button className="modal-close" onClick={() => setShowProjectManager(false)}>&times;</button>
                        <h3 style={{marginTop: 0, color: '#fb923c', marginBottom: '20px'}}>{t("projectManagement")}</h3>
                        
                        <div style={{maxHeight: '300px', overflowY: 'auto', marginBottom: '10px'}}>
                            {tempProjects.map((proj: any) => (
                                <div key={proj.id} style={{display: 'flex', alignItems: 'center', marginBottom: '10px', gap: '10px'}}>
                                    <input 
                                        type="text" 
                                        className="form-input" 
                                        value={proj.name}
                                        onChange={(e) => handleRenameTempProject(proj.id, e.target.value)}
                                        placeholder={t("projectName")}
                                        style={{flex: 1}}
                                    />
                                    {tempProjects.length > 1 && (
                                        <button 
                                            className="btn-link" 
                                            style={{color: '#ef4444', borderColor: '#ef4444', padding: '5px 10px', whiteSpace: 'nowrap'}}
                                            onClick={() => handleDeleteTempProject(proj.id)}
                                        >
                                            {t("delete")}
                                        </button>
                                    )}
                                </div>
                            ))}
                        </div>

                        {managerStatus && (
                            <div style={{color: '#ef4444', fontSize: '0.85rem', marginBottom: '15px', fontWeight: 500}}>
                                {managerStatus}
                            </div>
                        )}

                        <div style={{display: 'flex', flexDirection: 'column', gap: '10px'}}>
                            <button className="btn-primary" style={{width: '100%', background: 'transparent', border: '1px dashed #fb923c', color: '#fb923c'}} onClick={handleAddTempProject}>
                                {t("addNewProject")}
                            </button>
                            <button 
                                className="btn-primary" 
                                style={{width: '100%', opacity: managerStatus ? 0.5 : 1, cursor: managerStatus ? 'not-allowed' : 'pointer'}} 
                                onClick={saveProjectManagerChanges}
                                disabled={!!managerStatus}
                            >
                                {t("saveChanges")}
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {showUpdateModal && (
                <div className="modal-overlay" onClick={() => setShowUpdateModal(false)}>
                    <div className="modal-content" onClick={e => e.stopPropagation()} style={{width: '320px'}}>
                        <button className="modal-close" onClick={() => setShowUpdateModal(false)}>&times;</button>
                        <h3 style={{marginTop: 0, color: '#fb923c'}}>{t("checkUpdate")}</h3>
                        <div style={{margin: '20px 0', fontSize: '1rem'}}>
                            {updateResult?.has_update ? (
                                <div style={{color: '#10b981', fontWeight: 600}}>
                                    {t("updateAvailable")} {updateResult.latest_version}
                                </div>
                            ) : (
                                <div style={{color: '#6b7280'}}>
                                    {t("noUpdate")}
                                </div>
                            )}
                        </div>
                        <div style={{display: 'flex', gap: '10px'}}>
                            {updateResult?.has_update && (
                                <button 
                                    className="btn-primary" 
                                    style={{flex: 1}}
                                    onClick={() => {
                                        BrowserOpenURL("https://github.com/RapidAI/cceasy/releases");
                                        setShowUpdateModal(false);
                                    }}
                                >
                                    {t("downloadNow")}
                                </button>
                            )}
                            <button 
                                className="btn-primary" 
                                style={{flex: 1, backgroundColor: updateResult?.has_update ? '#6b7280' : '#fb923c'}}
                                onClick={() => setShowUpdateModal(false)}
                            >
                                {updateResult?.has_update ? t("hide") : "OK"}
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {showAbout && (
                <div className="modal-overlay" onClick={(e) => { if (e.target === e.currentTarget) setShowAbout(false); }}>
                    <div className="modal-content" onClick={e => e.stopPropagation()} style={{textAlign: 'center'}}>
                        <button className="modal-close" onClick={() => setShowAbout(false)}>&times;</button>
                        <img src={appIcon} alt="App Icon" style={{width: '64px', height: '64px', marginBottom: '15px', borderRadius: '12px', boxShadow: '0 4px 12px rgba(249, 115, 22, 0.2)'}} />
                        <h3 style={{marginTop: 0, color: '#fb923c'}}>Claude Code Easy Suite</h3>
                        <p style={{color: '#6b7280', margin: '5px 0'}}>Version V{APP_VERSION} Beta (Build {buildNumber})</p>
                        <p style={{color: '#6b7280', margin: '5px 0'}}>Author: Dr. Daniel</p>
                        <div style={{display: 'flex', flexDirection: 'column', gap: '10px', marginTop: '20px'}}>
                            <button 
                                className="btn-primary" 
                                onClick={() => BrowserOpenURL("https://github.com/RapidAI/cceasy")}
                                style={{display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px'}}
                            >
                                <span style={{fontSize: '1.2em'}}>GitHub</span>
                            </button>
                            <button 
                                className="btn-primary" 
                                onClick={() => BrowserOpenURL("https://github.com/RapidAI/cceasy/issues/new")}
                                style={{display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px', background: 'transparent', border: '1px solid #fb923c', color: '#fb923c'}}
                            >
                                {t("bugReport")}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    )
}

export default App