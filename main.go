package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath" // Remove since it's not used
	"strconv"
	"strings"
	"time"

	// 時間を使うために必要
	"github.com/BurntSushi/toml" // TOMLパッケージをインポート
)

// --- TOML構造体の定義 ---

// GameConfig はゲーム全体のTOML設定を表すルート構造体
type GameConfig struct {
	Nodes []Node `toml:"nodes"`
}

// Node はゲームの各ステップ（ノード）を表す構造体
type Node struct {
	ID       string    `toml:"id"`
	Type     string    `toml:"type"` // "story", "encounter", "end" など
	Text     string    `toml:"text"`
	Choices  []Choice  `toml:"choices,omitempty"`  // storyノードの選択肢
	Enemies  []*Enemy  `toml:"enemies,omitempty"`  // 戦闘の場合
	Outcomes []Outcome `toml:"outcomes,omitempty"` // ノードの結果

}

type Choice struct {
	// フィールド名を大文字に修正しました
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

// Outcome は遭遇戦の結果と次に進むノードを表す構造体
type Outcome struct {
	Description   string `toml:"description,omitempty"`
	Condition     string `toml:"condition,omitempty"` // "combat_won", "combat_lost" など
	Condition_int []int  `toml:"condition_int,omitempty"`
	NextNodeID    string `toml:"next_node_id"`
}

/*
type KaiDisciplines struct {
	Kamouflage     bool
	Hunting        bool
	SixthSense     bool
	Tracking       bool
	Healing        bool
	WeaponSkill    bool
	MindBlast      bool
	MindShield     bool
	AnimalKinship  bool
	MindOverMatter bool
}
*/

type Player struct {
	HP             int
	CS             int //combat skill
	GOLD           int
	MEAL           int
	KaiDisciplines map[string]bool //Kai Disciplines
	Weapon         []string        //Weapon
	Armor          []string        //Armor
	Items          []string        //Items
}

func init() {
	fmt.Println("Initializing CRT...")
	// crtマップを初期化
	crt = make(map[KeyPair]DamagePair)

	// TOMLファイルのパス
	filePath := filepath.Join(".", "combat_result_table.toml") // 実行ファイルと同じディレクトリを想定

	// TOMLファイルを読み込み
	var data CRTData
	if _, err := toml.DecodeFile(filePath, &data); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding TOML file: %v\n", err)
		// エラーが発生した場合、プログラムを終了させるか、デフォルト値を設定するなどの対応が必要
		os.Exit(1) // 例として終了
	}

	// 読み込んだデータをマップに格納
	for _, result := range data.Results {
		crt[result.KeyPair] = result.DamagePair
	}
	fmt.Println("CRT initialized successfully.")

}

// --- ゲームロジック ---

// GameState は現在のゲームの状態を保持する
type GameState struct {
	Player        *Player
	CurrentNodeID string
	Nodes         map[string]Node // IDでノードを検索するためのマップ
	Reader        *bufio.Reader   // ユーザー入力を受け取るためのリーダー
}

type KeyPair struct {
	RandNum  int `toml:"RandNum"`
	ComRatio int `toml:"ComRatio"`
}

type DamagePair struct {
	EnemyLoss  int  `toml:"EnemyLoss"`
	PlayerLoss int  `toml:"PlayerLoss"`
	IsKilled   bool `toml:"IsKilled"`
}

// CRTData はTOMLファイル全体の構造を定義します
type CRTData struct {
	Results []struct {
		KeyPair
		DamagePair
	} `toml:"results"`
}

// グローバル変数としてCRTマップを宣言
var crt map[KeyPair]DamagePair

func normalizeCombatRatio(ratio int) int {
	if ratio <= -11 {
		return -11 // -11以下はすべて-11として扱う
	}
	if ratio >= 11 {
		return 11 // 11以上はすべて11として扱う
	}
	return ratio // それ以外はそのまま
}

func contains_str(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func contains_int(slice []int, number int) bool {
	for _, i := range slice {
		if i == number {
			return true
		}
	}
	return false
}

// NewGameState は新しいGameStateを初期化する
func NewGameState(config *GameConfig) *GameState {
	nodeMap := make(map[string]Node) //ノードのマップを作成
	for _, node := range config.Nodes {
		nodeMap[node.ID] = node
	}
	return &GameState{ //GameState初期化
		Player: &Player{
			HP:   15,
			CS:   10,
			GOLD: 100,
			KaiDisciplines: map[string]bool{
				"Kamouflage":     false,
				"Hunting":        false,
				"SixthSense":     true,
				"Tracking":       false,
				"Healing":        false,
				"WeaponSkill":    false, //武器種が入る
				"MindBlast":      false,
				"MindShield":     false,
				"AnimalKinship":  false,
				"MindOverMatter": false,
			},
			Weapon: []string{"Sword"},
			Armor:  []string{"Leather"},
			Items:  []string{"Meal", "SilverKey"},
		},

		CurrentNodeID: config.Nodes[0].ID, // 最初のノードから開始
		Nodes:         nodeMap,
		Reader:        bufio.NewReader(os.Stdin),
	}
}

// Run はゲームループを開始する
func (gs *GameState) Run() {
	for {
		currentNode, exists := gs.Nodes[gs.CurrentNodeID]
		if !exists {
			fmt.Println("\nエラー: 存在しないノードIDに到達しました:", gs.CurrentNodeID)
			break
		}

		fmt.Println("\n---")
		fmt.Println(currentNode.Text)
		fmt.Println("---")

		switch currentNode.Type {
		case "story":
			gs.handleStoryNode(currentNode)
		case "encounter":
			gs.handleEncounterNode(currentNode)
		case "random_roll":
			gs.handleRandomNode(currentNode)
		case "end":
			fmt.Println("ゲーム終了。")
			return // ゲームループを終了
		default:
			fmt.Println("エラー: 未知のノードタイプ:", currentNode.Type)
			return
		}
	}
}

func (gs *GameState) handleRandomNode(node Node) {

	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	randomNumber := r.Intn(10)

	fmt.Printf("RandomNumberは%dです\n", randomNumber)

	fmt.Println("\n選択肢:")
	for i, choice := range node.Outcomes {
		fmt.Printf("%d. %s\n", i+1, choice.Description)
	}

	for {
		fmt.Print("選択してください (番号): ")
		input, _ := gs.Reader.ReadString('\n')
		input = strings.TrimSpace(input)
		choiceNum, err := strconv.Atoi(input)

		outcome := node.Outcomes[choiceNum-1]

		if err == nil &&
			contains_int(outcome.Condition_int, randomNumber) {
			gs.CurrentNodeID = outcome.NextNodeID
			break //RunLoopへ戻る
		} else {
			fmt.Println("条件を満たしていません。")
		}
	}
}

// handleStoryNode はストーリーノードの処理
func (gs *GameState) handleStoryNode(node Node) {
	if len(node.Choices) == 0 {
		fmt.Println("このノードには選択肢がありません。ゲーム終了。")
		gs.CurrentNodeID = "game_over" // 選択肢がなければゲームオーバーに送るか、別の処理
		return
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

		choice := node.Choices[choiceNum-1]
		//required_discipline_name := *choice.RequiredDiscipline
		//required_item_name := *choice.RequiredItem

		var required_discipline_name string
		if choice.RequiredDiscipline != nil {
			required_discipline_name = *choice.RequiredDiscipline
		}

		var required_item_name string
		if choice.RequiredItem != nil {
			required_item_name = *choice.RequiredItem
		}

		//fmt.Print(required_discipline_name)

		if err == nil && //エラーじゃなく
			choiceNum >= 1 && //1以上で
			choiceNum <= len(node.Choices) && //選択肢数以下で
			choice.RequiredDiscipline == nil && //必須ディシプリンなし
			choice.RequiredItem == nil { //必須アイテムなし
			gs.CurrentNodeID = node.Choices[choiceNum-1].NextNodeID
			break
		} else if err == nil &&
			choiceNum >= 1 &&
			choiceNum <= len(node.Choices) &&
			//choice.RequiredDiscipline != nil &&
			//choice.RequiredItem == nil &&
			gs.Player.KaiDisciplines[required_discipline_name] {
			gs.CurrentNodeID = node.Choices[choiceNum-1].NextNodeID
			break
		} else if err == nil &&
			choiceNum >= 1 &&
			choiceNum <= len(node.Choices) &&
			choice.RequiredDiscipline == nil &&
			choice.RequiredItem != nil &&
			contains_str(gs.Player.Items, required_item_name) {
			gs.CurrentNodeID = node.Choices[choiceNum-1].NextNodeID
			break
		} else {
			fmt.Println("無効な入力です。もう一度入力してください。")
		}
	}
}

func makeCombatResult(PCS int, ECS int) DamagePair {

	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	randomNumber := r.Intn(10)

	CombatRatio := PCS - ECS // 例えば、+5 の戦闘比率だったとする

	normalizedCR := normalizeCombatRatio(CombatRatio)

	key := KeyPair{RandNum: randomNumber, ComRatio: normalizedCR}
	result, ok := crt[key]

	if ok {
		return result
	} else {
		fmt.Println("Key not found in the map.")
		return DamagePair{
			EnemyLoss:  0,
			PlayerLoss: 0,
			IsKilled:   false,
		}
	}

}

// handleEncounterNode は遭遇戦ノードの処理 (簡易版)
func (gs *GameState) handleEncounterNode(node Node) {
	fmt.Println("\n--- エンカウント！ ---")

	for _, currentEnemy := range node.Enemies {
		// エンカウント情報が完全かチェックし、敵を設定

		for {
			fmt.Printf("\nLone Wolf (HP:%d CS;%d)",
				gs.Player.HP, gs.Player.CS)
			fmt.Printf("\n%s (HP:%d CS:%d)\n",
				currentEnemy.Name, currentEnemy.HP, currentEnemy.CS) // 敵のHPを更新して表示

			time.Sleep(1 * time.Second)
			fmt.Println("\n力を込めて物理で殴る！")
			time.Sleep(2 * time.Second)

			Edamage := makeCombatResult(gs.Player.CS, currentEnemy.CS).EnemyLoss
			Pdamage := makeCombatResult(gs.Player.CS, currentEnemy.CS).PlayerLoss
			currentEnemy.HP -= Edamage
			gs.Player.HP -= Pdamage
			fmt.Printf("あなたは%sに%dダメージを与えた！\nそしてあなたは%dダメージを受けた！\n",
				currentEnemy.Name, Edamage, Pdamage)

			// 敵のHPチェック
			if currentEnemy.HP <= 0 {
				fmt.Printf("%sを倒した！\n", currentEnemy.Name)
				// 勝利した場合の次のノードを探す

				break // 戦闘ループを終了し次の敵がいれば次の敵へ
			}

			if gs.Player.HP <= 0 {
				break // プレイヤーのHPが0以下になった場合、抜ける
			}

		}

		if gs.Player.HP <= 0 {
			fmt.Println("あなたは倒れた！")
			gs.CurrentNodeID = "game_over"
			break // プレイヤーのHPが0以下になった場合、ゲームオーバーへ
		}
	}
	foundOutcome := false
	for _, outcome := range node.Outcomes {
		if outcome.Condition == "combat_won" { // "combat_won" 条件をチェック
			gs.CurrentNodeID = outcome.NextNodeID
			foundOutcome = true
			break
		}
	}
	if !foundOutcome {
		fmt.Println("エラー: 勝利時の次のノードが見つかりません。ゲーム終了。")
		gs.CurrentNodeID = "game_over"
	}
}

func main() {

	// TOMLファイルを読み込む
	tomlData, err := ioutil.ReadFile("testlw.toml")
	if err != nil {
		log.Fatalf("Error reading TOML file: %v", err)
	}

	var config GameConfig // TOMLデータを格納する構造体のインスタンス

	// TOMLデータを構造体にデコードする
	if _, err := toml.Decode(string(tomlData), &config); err != nil {
		log.Fatalf("Error decoding TOML: %v", err)
	}
	// ゲームの状態を初期化し、ゲームを開始
	gameState := NewGameState(&config)
	gameState.Run()

}
