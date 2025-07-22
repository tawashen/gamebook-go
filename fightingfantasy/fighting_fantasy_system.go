package fightingfantasy

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"gamebook-go/game"
)

// FightingFantasySystem はFighting Fantasyゲームブックのルールを実装
type FightingFantasySystem struct {
	Rand *rand.Rand
}

// NewFightingFantasySystem は新しいFightingFantasySystemインスタンスを生成
func NewFightingFantasySystem() *FightingFantasySystem {
	return &FightingFantasySystem{
		Rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Initialize はFightingFantasySystemを初期化
func (ff *FightingFantasySystem) Initialize(config *game.GameConfig) error {
	fmt.Println("Fighting Fantasy initialized successfully.")
	return nil
}

// HandleNode はノードタイプに応じて処理
func (ff *FightingFantasySystem) HandleNode(gs *game.GameState, node game.Node) error {
	fmt.Println("\n---")
	fmt.Println(node.Text)
	fmt.Println("---")

	switch node.Type {
	case "story":
		return ff.handleStoryNode(gs, node)
	case "encounter":
		return ff.handleEncounterNode(gs, node)
	case "luck_test":
		return ff.handleLuckTestNode(gs, node)
	case "end":
		return nil
	default:
		return fmt.Errorf("unknown node type: %s", node.Type)
	}
}

// UpdatePlayer はプレイヤーの状態を更新
func (ff *FightingFantasySystem) UpdatePlayer(gs *game.GameState, action string) error {
	if action == "eat_meal" && contains(gs.Player.Inventory, "Meal") {
		gs.Player.Stats["STAMINA"] += 4
		gs.Player.Inventory = remove(gs.Player.Inventory, "Meal")
		fmt.Println("Ate a meal, restored 4 STAMINA!")
	}
	return nil
}

// handleStoryNode はストーリーノードを処理
func (ff *FightingFantasySystem) handleStoryNode(gs *game.GameState, node game.Node) error {
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
			//s.display_status()
			continue
		}

		choice := node.Choices[choiceNum-1]
		valid := true
		for key, condition := range choice.Conditions {
			if key == "item" && !contains(gs.Player.Inventory, condition) {
				valid = false
			} else if key == "skill" && strings.HasPrefix(condition, ">") {
				threshold, _ := strconv.Atoi(strings.TrimPrefix(condition, ">"))
				if gs.Player.Stats["SKILL"] <= threshold {
					valid = false
				}
			}
		}
		if valid {
			gs.CurrentNodeID = choice.NextNodeID
			break
		} else {
			//.display_status()
		}
	}
	return nil
}

// handleEncounterNode は戦闘ノードを処理
func (ff *FightingFantasySystem) handleEncounterNode(gs *game.GameState, node game.Node) error {
	fmt.Println("\n--- エンカウント！ ---")
	for _, enemy := range node.Enemies {
		for {
			fmt.Printf("\nYou (STAMINA:%d SKILL:%d)\n", gs.Player.Stats["STAMINA"], gs.Player.Stats["SKILL"])
			fmt.Printf("%s (STAMINA:%d SKILL:%d)\n", enemy.Name, enemy.HP, enemy.CS)

			time.Sleep(1 * time.Second)
			fmt.Println("\n攻撃！")
			time.Sleep(2 * time.Second)

			playerRoll := ff.Rand.Intn(6) + ff.Rand.Intn(6) + gs.Player.Stats["SKILL"]
			enemyRoll := ff.Rand.Intn(6) + ff.Rand.Intn(6) + enemy.CS
			fmt.Printf("あなた: %d vs 敵: %d\n", playerRoll, enemyRoll)

			if playerRoll > enemyRoll {
				enemy.HP -= 2
				fmt.Printf("あなたは%sに2ダメージを与えた！\n", enemy.Name)
			} else if playerRoll < enemyRoll {
				gs.Player.Stats["STAMINA"] -= 2
				fmt.Println("あなたは2ダメージを受けた！")
			} else {
				fmt.Println("引き分け！")
			}

			if enemy.HP <= 0 {
				fmt.Printf("%sを倒した！\n", enemy.Name)
				break
			}
			if gs.Player.Stats["STAMINA"] <= 0 {
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

// handleLuckTestNode はLuckテストノードを処理
func (ff *FightingFantasySystem) handleLuckTestNode(gs *game.GameState, node game.Node) error {
	roll := ff.Rand.Intn(6) + ff.Rand.Intn(6)
	fmt.Printf("Luckテスト: 2D6 = %d (LUCK以下で成功: %d)\n", roll, gs.Player.Stats["LUCK"])

	gs.Player.Stats["LUCK"] -= 1 // LuckテストごとにLUCKを1減らす
	if gs.Player.Stats["LUCK"] < 0 {
		gs.Player.Stats["LUCK"] = 0
	}

	for _, outcome := range node.Outcomes {
		if outcome.Condition == "luck_success" && roll <= gs.Player.Stats["LUCK"] {
			gs.CurrentNodeID = outcome.NextNodeID
			return nil
		} else if outcome.Condition == "luck_failure" && roll > gs.Player.Stats["LUCK"] {
			gs.CurrentNodeID = outcome.NextNodeID
			return nil
		}
	}
	gs.CurrentNodeID = "game_over"
	return fmt.Errorf("no valid outcome found")
}

// contains はスライスに指定された文字列が含まれるか確認
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// remove はスライスから指定された文字列を削除
func remove(slice []string, str string) []string {
	result := []string{}
	for _, s := range slice {
		if s != str {
			result = append(result, s)
		}
	}
	return result
}
