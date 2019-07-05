package Structs

import "github.com/faiface/pixel"

type Spaceship struct {
	Shipsprite *pixel.Sprite
	X          float64
	Y          float64
}

type Monster struct {
	Monstersprite1 *pixel.Sprite
	Monstersprite2 *pixel.Sprite
	Id             byte
	MonsterType    int
	Spritebool     bool
	Movingright    bool
	X              float64
	Y              float64
}

type Shot struct {
	Shotsprite  *pixel.Sprite
	MonsterShot bool
	X           float64
	Y           float64
}

type Client struct {
	IP   string
	Port int
}

type ClientData struct {
	ClientPortBind string
	Datatype       string
	Data           interface{}
}

type ServerData struct {
	Datatype string
	Data     interface{}
}
