package lonewolf

import (
	"fmt"
	"math/rand"
	"time"

	"gamebook-go/game"

	"github.com/BurntSushi/toml"
)

// KeyPair は戦闘結果テーブルのキーを定義
type KeyPair struct {
	RandNum  int `toml:"RandNum"`
	ComRatio int `toml:"ComRatio"`
}

// DamagePair は戦闘結果テーブルの値を定義
type DamagePair struct {
	EnemyLoss  int  `toml:"EnemyLoss"`
	PlayerLoss int  `toml:"PlayerLoss"`
	IsKilled   bool `toml:"IsKilled"`
}

// CRTData はTOMLファイル全体の構造を定義
type CRTData struct {
	Results []struct {
		KeyPair
		DamagePair
	} `toml:"results"`
}

// LoneWolfSystem はLone Wolfゲームブックのルールを実装
type LoneWolfSystem struct {
	CRT       map[KeyPair]DamagePair
	Rand      *rand.Rand
	CRTFile   string
	ConfigDir string
}

// NewLoneWolfSystem は新しいLoneWolfSystemインスタンスを生成
func NewLoneWolfSystem(crtFile string) *LoneWolfSystem {
	return &LoneWolfSystem{
		CRT:       make(map[KeyPair]DamagePair),
		Rand:      rand.New(rand.NewSource(time.Now().UnixNano())),
		CRTFile:   crtFile,
		ConfigDir: ".",
	}
}

// Initialize はLoneWolfSystemを初期化
func (lw *LoneWolfSystem) Initialize(config *game.GameConfig) error {
	stats, ok := config.Player["stats"].(map[string]interface{})
	if !ok || stats["HP"] == nil || stats["CS"] == nil {
		return fmt.Errorf("missing HP or CS in player stats")
	}

	var data CRTData
	if _, err := toml.DecodeFile(lw.CRTFile, &data); err != nil {
		return fmt.Errorf("error decoding CRT file: %v", err)
	}

	for _, result := range data.Results {
		lw.CRT[result.KeyPair] = result.DamagePair
	}

	fmt.Println("Lone Wolf CRT initialized successfully.")
	return nil
}

// HandleNode はノードタイプに応じて処理
func (lw *LoneWolfSystem) HandleNode(gs *game.GameState, node game.Node) error {
	// 以前のコードをそのまま使用（game. を付けて共用構造体を参照）
	// ...（HandleNode, handleStoryNode, handleEncounterNode, handleRandomNode の実装）
	return nil
}

// UpdatePlayer はプレイヤーの状態を更新
func (lw *LoneWolfSystem) UpdatePlayer(gs *game.GameState, action string) error {
	if action == "heal" && gs.Player.Attributes["Healing"] {
		gs.Player.Stats["HP"] += 1
		fmt.Println("Healing Discipline restored 1 HP!")
	}
	return nil
}

// ...（他のメソッド：makeCombatResult, normalizeCombatRatio, contains, containsInt）
