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
	ID        string     `toml:"id"`
	Type      string     `toml:"type"` // "story", "encounter", "end" など
	Text      string     `toml:"text"`
	Choices   []Choice   `toml:"choices,omitempty"`   // storyノードの選択肢
	Encounter *Encounter `toml:"encounter,omitempty"` // encounterノードの詳細
	Outcomes  []Outcome  `toml:"outcomes,omitempty"`  // encounterノードの結果
}

// Choice はユーザーが選択できる項目を表す構造体
type Choice struct {
	Text       string `toml:"text"`
	NextNodeID string `toml:"next_node_id"`
}

// Encounter は遭遇戦の詳細を表す構造体
type Encounter struct {
	Type             string      `toml:"type"`               // "combat" など
	CombatSystemType string      `toml:"combat_system_type"` // "DnD5e" など
	CombatData       *CombatData `toml:"combat_data,omitempty"`
}

// CombatData は戦闘の詳細データ
type CombatData struct {
	Terrain    string  `toml:"terrain"`
	Difficulty string  `toml:"difficulty"`
	Enemies    []Enemy `toml:"enemies"`
}

// Enemy は戦闘の敵キャラクター
type Enemy struct {
	Name  string `toml:"name"`
	Stats *Stats `toml:"stats"`
}

// Stats は敵のステータス
type Stats struct {
	HP int `toml:"hp"`
	AC int `toml:"ac"`
}

// Outcome は遭遇戦の結果と次に進むノードを表す構造体
type Outcome struct {
	Condition  string `toml:"condition"` // "combat_won", "combat_lost" など
	NextNodeID string `toml:"next_node_id"`
}

/*
 type PlayerStats {
	HP int
	CS int
	GOLD int

 }
*/

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

// NewGameState は新しいGameStateを初期化する
func NewGameState(config *GameConfig) *GameState {
	nodeMap := make(map[string]Node)
	for _, node := range config.Nodes {
		nodeMap[node.ID] = node
	}
	return &GameState{
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
		case "end":
			fmt.Println("ゲーム終了。")
			return // ゲームループを終了
		default:
			fmt.Println("エラー: 未知のノードタイプ:", currentNode.Type)
			return
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
		fmt.Printf("%d. %s\n", i+1, choice.Text)
	}

	for {
		fmt.Print("選択してください (番号): ")
		input, _ := gs.Reader.ReadString('\n')
		input = strings.TrimSpace(input)
		choiceNum, err := strconv.Atoi(input)

		if err == nil && choiceNum >= 1 && choiceNum <= len(node.Choices) {
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

	// プレイヤーの簡易ステータス（ここでは固定値）
	playerHP := 15
	playerAC := 15

	//var currentEnemy *Enemy // 現在の敵へのポインタを保持する変数

	for _, currentEnemy := range node.Encounter.CombatData.Enemies {
		// エンカウント情報が完全かチェックし、敵を設定

		fmt.Printf("戦闘システム: %s, 難易度: %s\n",
			node.Encounter.CombatSystemType,
			node.Encounter.CombatData.Difficulty)
		fmt.Printf("敵: %s (HP:%d AC:%d)\n",
			currentEnemy.Name, currentEnemy.Stats.HP, currentEnemy.Stats.AC)

		for {
			fmt.Printf("\n%s (HP:%d AC:%d)\n",
				currentEnemy.Name, currentEnemy.Stats.HP, currentEnemy.Stats.AC) // 敵のHPを更新して表示

			// プレイヤーの選択肢を表示
			fmt.Println("選択してください (番号): ")
			fmt.Println("1. 力を込めて物理で殴る")
			fmt.Println("2. 心を鎮めて魔法を唱える")
			fmt.Println("3. 懐を探って道具を使う")

			input, _ := gs.Reader.ReadString('\n')
			input = strings.TrimSpace(input)
			choiceNum, err := strconv.Atoi(input)

			if err != nil || choiceNum < 1 || choiceNum > 3 { // 入力エラーまたは範囲外の場合
				fmt.Println("無効な入力です。1〜3で選択してください。")
				continue // ループの最初に戻る
			}

			// 選択肢に応じた処理
			switch choiceNum {
			case 1: // 物理攻撃
				//	damage := playerAC - currentEnemy.Stats.AC // 簡易的にプレイヤーの攻撃力をそのままダメージとする
				//	currentEnemy.Stats.HP -= damage
				Edamage := makeCombatResult(playerAC, currentEnemy.Stats.AC).EnemyLoss
				Pdamage := makeCombatResult(playerAC, currentEnemy.Stats.AC).PlayerLoss
				currentEnemy.Stats.HP -= Edamage
				playerHP -= Pdamage
				fmt.Printf("あなたは%sに%dダメージを与えた！\nそしてあなたは%dダメージを受けた！\n",
					currentEnemy.Name, Edamage, Pdamage)
			case 2: // 魔法
				fmt.Println("しかし何も起きなかった。")
			case 3: // 道具
				fmt.Println("何も持っていない。")
			}

			// 敵のHPチェック
			if currentEnemy.Stats.HP <= 0 {
				fmt.Printf("%sを倒した！\n", currentEnemy.Name)
				// 勝利した場合の次のノードを探す
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
				break // 戦闘ループを終了し、次のノードへ
			}

			if playerHP <= 0 {
				fmt.Println("あなたは倒れた！")
				gs.CurrentNodeID = "game_over"
				break // プレイヤーのHPが0以下になった場合、ゲームオーバーへ
			}

		}
	}
}

func main() {

	/*
		source := rand.NewSource(time.Now().UnixNano())
		r := rand.New(source)


		// --- 使い方例 ---
		// 1. 乱数を振る (0-9)
		randomNumber := r.Intn(10)

		// 2. Combat Ratioを計算する (例としてここでは固定値)
		// 実際のゲームでは、プレイヤーのCOMBAT SKILLと敵のCOMBAT SKILLの差などから計算されます
		exampleCombatRatio := 5 // 例えば、+5 の戦闘比率だったとする

		// 3. Combat Ratioを正規化する
		normalizedCR := normalizeCombatRatio(exampleCombatRatio)

		// 4. マップから結果を取得する
		key := KeyPair{RandNum: randomNumber, ComRatio: normalizedCR}
		result, ok := crt[key]

		if ok {
			fmt.Printf("Random Number: %d, Combat Ratio: %d, EnemyLoss: %d, PlayerLoss: %d\n",
				randomNumber, exampleCombatRatio, result.EnemyLoss, result.PlayerLoss)
		} else {
			fmt.Println("Key not found in the map.")
		}

	*/
	// TOMLファイルを読み込む
	tomlData, err := ioutil.ReadFile("game.toml")
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
