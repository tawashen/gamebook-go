package game

import "bufio"

// GameConfig はゲーム全体のTOML設定を表す
type GameConfig struct {
	System string                 `toml:"system"`
	Player map[string]interface{} `toml:"player"`
	Nodes  []Node                 `toml:"nodes"`
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
	Description string            `toml:"description"`
	NextNodeID  string            `toml:"next_node_id"`
	Conditions  map[string]string `toml:"conditions,omitempty"`
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
	Stats      map[string]int
	Attributes map[string]bool
	Inventory  []string
	Equipment  map[string]string
}

// GameState はゲームの状態を保持
type GameState struct {
	Player        *Player
	CurrentNodeID string
	Nodes         map[string]Node
	Reader        *bufio.Reader
}
