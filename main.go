package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

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

// --- ゲームロジック ---

// GameState は現在のゲームの状態を保持する
type GameState struct {
	CurrentNodeID string
	Nodes         map[string]Node // IDでノードを検索するためのマップ
	Reader        *bufio.Reader   // ユーザー入力を受け取るためのリーダー
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

// handleEncounterNode は遭遇戦ノードの処理 (簡易版)
func (gs *GameState) handleEncounterNode(node Node) {
	fmt.Println("\n--- エンカウント！ ---")

	var currentEnemy *Enemy // 現在の敵へのポインタを保持する変数

	// エンカウント情報が完全かチェックし、敵を設定
	if node.Encounter != nil && node.Encounter.Type == "combat" &&
		node.Encounter.CombatData != nil && len(node.Encounter.CombatData.Enemies) > 0 {

		// 敵は常に1体という前提なので、最初の敵を取得
		currentEnemy = &node.Encounter.CombatData.Enemies[0]

		fmt.Printf("戦闘システム: %s, 難易度: %s\n",
			node.Encounter.CombatSystemType,
			node.Encounter.CombatData.Difficulty)
		fmt.Printf("敵: %s (HP:%d AC:%d)\n",
			currentEnemy.Name, currentEnemy.Stats.HP, currentEnemy.Stats.AC)

	} else {
		fmt.Println("エンカウント情報が不完全です。または敵がいません。ゲーム終了。")
		gs.CurrentNodeID = "game_over" // 不完全ならゲームオーバーへ
		return                         // 関数を終了
	}

	// プレイヤーの簡易ステータス（ここでは固定値）
	playerAtk := 15
	// playerHP := 15 // 現在のロジックでは使用されないためコメントアウト
	// playerDef := 15 // 現在のロジックでは使用されないためコメントアウト

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
			damage := playerAtk - currentEnemy.Stats.AC // 簡易的にプレイヤーの攻撃力をそのままダメージとする
			currentEnemy.Stats.HP -= damage
			fmt.Printf("あなたは%sに%dダメージを与えた！\n", currentEnemy.Name, damage)
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

		// (敵の攻撃など、ターン制の処理を追加する場合はここに記述)
		// 現状は敵の攻撃はないため、敵のHPが0にならない限りループが続く
	}
}

/*
// handleEncounterNode は遭遇戦ノードの処理 (簡易版)
func (gs *GameState) handleEncounterNode(node Node) {
	fmt.Println("\n--- エンカウント！ ---")
	if node.Encounter != nil && node.Encounter.Type == "combat" {
		fmt.Printf("戦闘システム: %s, 難易度: %s\n",
			node.Encounter.CombatSystemType,
			node.Encounter.CombatData.Difficulty)
		for _, enemy := range node.Encounter.CombatData.Enemies {
			fmt.Printf("敵: %s (HP:%d AC:%d)\n", enemy.Name, enemy.Stats.HP, enemy.Stats.AC)
		}
	} else {
		fmt.Println("エンカウント情報が不完全です。")
	}

	// 簡易的な選択: 戦闘の勝利か敗北を選ぶ
	//fmt.Println("\n戦闘の結果を選んでください:")
	//fmt.Println("1. 勝利する (combat_won)")
	//fmt.Println("2. 敗北する (combat_lost)")

	for {
		fmt.Print("選択してください (番号): ")
		fmt.Print("1.力を込めて物理で殴る")
		fmt.Print("2.心を鎮めて魔法を唱える")
		fmt.Print("3.懐を探って道具を使う")
		input, _ := gs.Reader.ReadString('\n')
		input = strings.TrimSpace(input)
		choiceNum, err := strconv.Atoi(input)

		if err == nil && (choiceNum >= 1 && choiceNum <= 3) {
			//var chosenCondition string
			var playerAtk int := 15
			var playerHP int := 15
			var playerDef int := 15
			if choiceNum == 1 {
				enemy.Stats.HP = enemy.Stats.HP - playerAtk
			}
			elseif choiceNum == 2 {
				fmt.Print("しかし何も起きなかった")
			}
			elseif choiceNum == 3 {
				fmt.Print("何も持っていない")
			}
			else {
				fmt.Print("無効な入力です。もう一度入力してください。")
			}}

			// 選択された結果に対応する次のノードを探す
			/*
			foundOutcome := false
			for _, outcome := range node.Outcomes {
				if outcome.Condition == chosenCondition {
					gs.CurrentNodeID = outcome.NextNodeID
					foundOutcome = true
					break
				}
			}
			 if enemy.Stats.HP <= 0 {
				gs.nodes.outcome = "combat_won"
			}

			if !foundOutcome {
				fmt.Println("エラー: 選択された結果に対応する次のノードが見つかりません。ゲーム終了。")
				gs.CurrentNodeID = "game_over"
			}
			break
		} else {
			fmt.Println("無効な入力です。もう一度入力してください。")
		}
	}
}
*/

func main() {
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
