package lonewolf

import (
	"bufio"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

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

// GameConfig はゲーム全体のTOML設定を表す
type GameConfig struct {
	System string `toml:"system"`
	Nodes  []Node `toml:"nodes"`
}

// Node はゲームの各ステップ（ノード）を表す
type Node struct {
	ID       string    `toml:"id"`
	Type     string    `toml:"type"`
	Text     string    `toml:"text"`
	Choices  []Choice  `toml:"choices,omitempty"`
	Enemies  []*Enemy  `toml:"enemies,omitempty"`
	Outcomes []Outcome `toml:"outcomes,omitempty"`
}

// Choice は選択肢を表す
type Choice struct {
	Description        string  `toml:"description"`
	NextNodeID         string  `toml:"next_node_id"`
	RequiredDiscipline *string `toml:"required_discipline,omitempty"`
	RequiredItem       *string `toml:"required_item,omitempty"`
}

// Enemy は戦闘の敵キャラクター
type Enemy struct {
	Name string `toml:"Name"`
	HP   int    `toml:"HP"`
	CS   int    `toml:"CS"`
}

// Outcome は遭遇戦の結果と次に進むノードを表す
type Outcome struct {
	Description  string `toml:"description,omitempty"`
	Condition    string `toml:"condition,omitempty"`
	ConditionInt []int  `toml:"condition_int,omitempty"`
	NextNodeID   string `toml:"next_node_id"`
}

// Player はプレイヤーの状態を表す
type Player struct {
	HP             int
	CS             int
	GOLD           int
	MEAL           int
	KaiDisciplines map[string]bool
	Weapon         []string
	Armor          []string
	Items          []string
}

// GameState はゲームの状態を保持
type GameState struct {
	Player        *Player
	CurrentNodeID string
	Nodes         map[string]Node
	Reader        *bufio.Reader
}

// LoneWolfSystem はLone Wolfゲームブックのルールを実装
type LoneWolfSystem struct {
	CRT       map[KeyPair]DamagePair // 戦闘結果テーブル
	Rand      *rand.Rand             // 乱数生成器
	CRTFile   string                 // 戦闘結果テーブルのファイルパス
	ConfigDir string                 // 追加の設定ファイルディレクトリ
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
func (lw *LoneWolfSystem) Initialize(config *GameConfig) error {
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
func (lw *LoneWolfSystem) HandleNode(gs *GameState, node Node) error {
	fmt.Println("\n---")
	fmt.Println(node.Text)
	fmt.Println("---")

	switch node.Type {
	case "story":
		return lw.handleStoryNode(gs, node)
	case "encounter":
		return lw.handleEncounterNode(gs, node)
	case "random_roll":
		return lw.handleRandomNode(gs, node)
	case "end":
		return nil
	default:
		return fmt.Errorf("unknown node type: %s", node.Type)
	}
}

// handleStoryNode はストーリーノードを処理
func (lw *LoneWolfSystem) handleStoryNode(gs *GameState, node Node) error {
	if len(node.Choices) == 0 {
		gs.CurrentNodeID = "game_over"
		return fmt.Errorf("no choices available")
	}

	fmt.Println("\n選択肢:")
	for i, choice := range node.Choices {
		fmt.Printf("%d. %s\n", i+1, choice.Description)
	}

	for {
		fmt.Print("選択してください (番号): ")
		input, _ := gs.Reader.ReadString('\n')
		input = strings.TrimSpace(input)
		choiceNum, err := strconv.Atoi(input)
		if err != nil || choiceNum < 1 || choiceNum > len(node.Choices) {
			gs.display_status()
			continue
		}

		choice := node.Choices[choiceNum-1]
		requiredDiscipline := ""
		if choice.RequiredDiscipline != nil {
			requiredDiscipline = *choice.RequiredDiscipline
		}
		requiredItem := ""
		if choice.RequiredItem != nil {
			requiredItem = *choice.RequiredItem
		}

		if choice.RequiredDiscipline == nil && choice.RequiredItem == nil {
			gs.CurrentNodeID = choice.NextNodeID
			break
		} else if choice.RequiredDiscipline != nil && gs.Player.KaiDisciplines[requiredDiscipline] {
			gs.CurrentNodeID = choice.NextNodeID
			break
		} else if choice.RequiredItem != nil && contains_str(gs.Player.Items, requiredItem) {
			gs.CurrentNodeID = choice.NextNodeID
			break
		} else {
			gs.display_status()
		}
	}
	return nil
}

// handleEncounterNode は戦闘ノードを処理
func (lw *LoneWolfSystem) handleEncounterNode(gs *GameState, node Node) error {
	fmt.Println("\n--- エンカウント！ ---")
	for _, enemy := range node.Enemies {
		for {
			fmt.Printf("\nLone Wolf (HP:%d CS:%d)\n", gs.Player.HP, gs.Player.CS)
			fmt.Printf("%s (HP:%d CS:%d)\n", enemy.Name, enemy.HP, enemy.CS)

			time.Sleep(1 * time.Second)
			fmt.Println("\n力を込めて物理で殴る！")
			time.Sleep(2 * time.Second)

			result := lw.makeCombatResult(gs.Player.CS, enemy.CS)
			enemy.HP -= result.EnemyLoss
			gs.Player.HP -= result.PlayerLoss
			fmt.Printf("あなたは%sに%dダメージを与えた！\nそしてあなたは%dダメージを受けた！\n",
				enemy.Name, result.EnemyLoss, result.PlayerLoss)

			if enemy.HP <= 0 {
				fmt.Printf("%sを倒した！\n", enemy.Name)
				break
			}
			if gs.Player.HP <= 0 {
				gs.CurrentNodeID = "game_over"
				return fmt.Errorf("player defeated")
			}
		}
	}

	for _, outcome := range node.Outcomes {
		if outcome.Condition == "combat_won" {
			gs.CurrentNodeID = outcome.NextNodeID
			return nil
		}
	}
	gs.CurrentNodeID = "game_over"
	return fmt.Errorf("no combat_won outcome found")
}

// handleRandomNode はランダムノードを処理
func (lw *LoneWolfSystem) handleRandomNode(gs *GameState, node Node) error {
	randomNumber := lw.Rand.Intn(10)
	fmt.Printf("RandomNumberは%dです\n", randomNumber)

	fmt.Println("\n選択肢:")
	for i, outcome := range node.Outcomes {
		fmt.Printf("%d. %s\n", i+1, outcome.Description)
	}

	for {
		fmt.Print("選択してください (番号): ")
		input, _ := gs.Reader.ReadString('\n')
		input = strings.TrimSpace(input)
		choiceNum, err := strconv.Atoi(input)
		if err != nil || choiceNum < 1 || choiceNum > len(node.Outcomes) {
			gs.display_status()
			continue
		}

		outcome := node.Outcomes[choiceNum-1]
		if contains_int(outcome.ConditionInt, randomNumber) {
			gs.CurrentNodeID = outcome.NextNodeID
			break
		} else {
			fmt.Println("条件を満たしていません。")
			gs.display_status()
		}
	}
	return nil
}

// makeCombatResult は戦闘結果を計算
func (lw *LoneWolfSystem) makeCombatResult(playerCS, enemyCS int) DamagePair {
	combatRatio := playerCS - enemyCS
	normalizedCR := normalizeCombatRatio(combatRatio)
	randomNumber := lw.Rand.Intn(10)

	key := KeyPair{RandNum: randomNumber, ComRatio: normalizedCR}
	result, ok := lw.CRT[key]
	if !ok {
		fmt.Println("Key not found in CRT map.")
		return DamagePair{EnemyLoss: 0, PlayerLoss: 0, IsKilled: false}
	}
	return result
}

// normalizeCombatRatio は戦闘比率を正規化
func normalizeCombatRatio(ratio int) int {
	if ratio <= -11 {
		return -11
	}
	if ratio >= 11 {
		return 11
	}
	return ratio
}

// contains_str はスライスに指定された文字列が含まれるか確認
func contains_str(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// contains_int はスライスに指定された整数が含まれるか確認
func contains_int(slice []int, number int) bool {
	for _, i := range slice {
		if i == number {
			return true
		}
	}
	return false
}
