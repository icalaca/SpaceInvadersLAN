package main

//go get github.com/faiface/pixel
//cd $GOPATH/src/golang.org/x/ && git clone https://github.com/golang/image.git
//go get github.com/faiface/glhf
//go get github.com/go-gl/glfw/v3.2/glfw
import (
	"bytes"
	"container/list"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"math"
	"os"
	"strconv"
	"sync"
	"time"

	"./BEBcast"
	"./PP2PLink"
	"./Structs"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font/basicfont"
)

var sprites, err = loadPicture("sprites/sprites.png")
var PortBind = ""
var serverShipAddr = []string{"127.0.0.1:4002"}
var serverMainAddr = []string{"127.0.0.1:4003"}
var monsterMutex = &sync.Mutex{}
var shipMutex = &sync.Mutex{}
var shotMutex = &sync.Mutex{}
var mShotMutex = &sync.Mutex{}
var winW = 400.0
var winH = 300.0

type Spaceship Structs.Spaceship
type Monster Structs.Monster
type Shot Structs.Shot
type ClientData Structs.ClientData
type ServerData Structs.ServerData

func CreateSpaceship() *Spaceship {
	ship := Spaceship{}
	ship.Shipsprite = pixel.NewSprite(sprites, pixel.R(20, sprites.Bounds().H()-60, 38, sprites.Bounds().H()-47))
	ship.X = 0
	ship.Y = 0
	return &ship
}

func CreateMonster(x float64, y float64, mSprite bool, mMovingRight bool, monstertype float64, id byte) *Monster {
	monster := Monster{}
	switch monstertype {
	case 0:
		monster.Monstersprite1 = pixel.NewSprite(sprites, pixel.R(3, sprites.Bounds().H()-41, 17, sprites.Bounds().H()-27))
		monster.Monstersprite2 = pixel.NewSprite(sprites, pixel.R(23, sprites.Bounds().H()-41, 37, sprites.Bounds().H()-27))
	case 1:
		monster.Monstersprite1 = pixel.NewSprite(sprites, pixel.R(1, sprites.Bounds().H()-14, 19, sprites.Bounds().H()))
		monster.Monstersprite2 = pixel.NewSprite(sprites, pixel.R(21, sprites.Bounds().H()-14, 39, sprites.Bounds().H()))
	case 2:
		monster.Monstersprite1 = pixel.NewSprite(sprites, pixel.R(0, sprites.Bounds().H()-27, 20, sprites.Bounds().H()-14))
		monster.Monstersprite2 = pixel.NewSprite(sprites, pixel.R(20, sprites.Bounds().H()-27, 40, sprites.Bounds().H()-14))
	}
	monster.Id = id
	monster.X = x
	monster.Y = y
	monster.Spritebool = mSprite
	monster.Movingright = mMovingRight
	return &monster
}

func CreateShot(shipX float64, shipY float64, mShot bool) *Shot {
	shot := Shot{}
	if mShot {
		shot.Shotsprite = pixel.NewSprite(sprites, pixel.R(0, sprites.Bounds().H()-79, 5, sprites.Bounds().H()-70))
	} else {
		shot.Shotsprite = pixel.NewSprite(sprites, pixel.R(20, sprites.Bounds().H()-66, 23, sprites.Bounds().H()-60))
	}
	shot.MonsterShot = mShot
	shot.X = shipX
	shot.Y = shipY
	return &shot
}

func (ship *Spaceship) Draw(win *pixelgl.Window) {
	ship.Shipsprite.Draw(win, pixel.IM.Moved(pixel.V(ship.X, ship.Y)))
}

func (monster *Monster) Draw(win *pixelgl.Window) {
	if monster.Spritebool {
		monster.Monstersprite1.Draw(win, pixel.IM.Moved(pixel.V(monster.X, monster.Y)))
	} else {
		monster.Monstersprite2.Draw(win, pixel.IM.Moved(pixel.V(monster.X, monster.Y)))
	}
}

func (shot *Shot) Draw(win *pixelgl.Window) {
	shot.Shotsprite.Draw(win, pixel.IM.Moved(pixel.V(shot.X, shot.Y)))
	if shot.MonsterShot {
		shot.Y--
	} else {
		shot.Y++
	}
}

func loadPicture(path string) (pixel.Picture, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return pixel.PictureDataFromImage(img), nil
}

func (ship *Spaceship) initShip() {
	ship.X = winW / 2
	ship.Y = ship.Shipsprite.Frame().H() / 2
}

//Auxilia no envio dos dados pro servidor
func sendData(beb *BEBcast.BestEffortBroadcast_Module, data interface{}, datatype string) {
	var req BEBcast.BestEffortBroadcast_Req_Message
	cData := ClientData{PortBind, datatype, data}
	switch datatype {
	case "PortHandShake", "Shot", "KillMonster", "KillShip", "CloseConnection":
		req = BEBcast.BestEffortBroadcast_Req_Message{
			Addresses: serverMainAddr,
			Message:   cData}
	default:
		req = BEBcast.BestEffortBroadcast_Req_Message{
			Addresses: serverShipAddr,
			Message:   cData}
	}
	beb.Req <- req
}

func run() {
	cfg := pixelgl.WindowConfig{
		Title:  "Space Invaders",
		Bounds: pixel.R(0, 0, winW, winH),
	}
	win, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}
	wonStr := "YOU WON!"
	gameoverStr := "GAME OVER!"
	win.Clear(colornames.Black)
	monsters := list.New()
	shots := list.New()
	mShots := list.New()
	port := 0

	shipDict := make(map[string]interface{})
	ship := CreateSpaceship()
	ship.initShip()
	dead := false
	won := false
	gameover := false
	gamestarted := false
	basicAtlas := text.NewAtlas(basicfont.Face7x13, text.ASCII)
	wonTxt := text.New(pixel.V(winW/2, winH/2), basicAtlas)
	wonTxt.Dot.X -= math.Round(wonTxt.BoundsOf(wonStr).W() / 2)
	gameoverTxt := text.New(pixel.V(winW/2, winH/2), basicAtlas)
	gameoverTxt.Dot.X -= math.Round(gameoverTxt.BoundsOf(gameoverStr).W() / 2)
	fmt.Fprintln(wonTxt, wonStr)
	fmt.Fprintln(gameoverTxt, gameoverStr)
	//Procura por uma sequencia de 3 portas disponiveis
	for i := 5000; i < 6000; i++ {
		if PP2PLink.Check("127.0.0.1:"+strconv.Itoa(i)) &&
			PP2PLink.Check("127.0.0.1:"+strconv.Itoa(i+1)) &&
			PP2PLink.Check("127.0.0.1:"+strconv.Itoa(i+2)) {
			port = i
			break
		}
	}
	if port == 0 {
		os.Exit(1)
	}
	PortBind = strconv.Itoa(port)
	bebMonsters := BEBcast.BestEffortBroadcast_Module{
		Req: make(chan BEBcast.BestEffortBroadcast_Req_Message),
		Ind: make(chan BEBcast.BestEffortBroadcast_Ind_Message)}
	bebShip := BEBcast.BestEffortBroadcast_Module{
		Req: make(chan BEBcast.BestEffortBroadcast_Req_Message),
		Ind: make(chan BEBcast.BestEffortBroadcast_Ind_Message)}
	bebOthers := BEBcast.BestEffortBroadcast_Module{
		Req: make(chan BEBcast.BestEffortBroadcast_Req_Message),
		Ind: make(chan BEBcast.BestEffortBroadcast_Ind_Message)}

	bebMonsters.Init("127.0.0.1:"+PortBind, 8192)
	bebShip.Init("127.0.0.1:"+strconv.Itoa(port+1), 1024)
	bebOthers.Init("127.0.0.1:"+strconv.Itoa(port+2), 1024)
	//envia os dados(ip e porta) do cliente para o servidor
	//Apenas a primeira porta da sequencia é enviada para o servidor
	//de forma que o servidor sabe que bebShip recebe pela porta Port+1 e
	//bebOthers pela porta Port+2
	sendData(&bebOthers, nil, "PortHandShake")

	go func() {
		for {
			in := <-bebOthers.Ind
			sData := ServerData{}
			reqBodyBytes := new(bytes.Buffer)
			json.NewDecoder(reqBodyBytes).Decode(in.Message)
			json.Unmarshal([]byte(in.Message.(string)), &sData)
			switch sData.Datatype {
			//Recebe o tiro efetuado por algum cliente e coloca na lista de tiros
			case "ShotDict":
				data := sData.Data.(map[string]interface{})
				shotMutex.Lock()
				shots.PushFront(CreateShot(data["X"].(float64), data["Y"].(float64), false))
				shotMutex.Unlock()
			}
		}
	}()

	go func() {
		for {
			in := <-bebShip.Ind
			sData := ServerData{}
			reqBodyBytes := new(bytes.Buffer)
			json.NewDecoder(reqBodyBytes).Decode(in.Message)
			json.Unmarshal([]byte(in.Message.(string)), &sData)
			switch sData.Datatype {
			//Recebe a lista com as naves dos clientes
			case "ShipDict":
				if !gamestarted {
					gamestarted = true
				}
				data := sData.Data.(map[string]interface{})
				shipMutex.Lock()
				shipDict = data
				shipMutex.Unlock()
			}
		}
	}()

	go func() {
		for {
			in := <-bebMonsters.Ind
			sData := ServerData{}
			reqBodyBytes := new(bytes.Buffer)
			json.NewDecoder(reqBodyBytes).Decode(in.Message)
			json.Unmarshal([]byte(in.Message.(string)), &sData)
			switch sData.Datatype {
			//Recebe a lista de monstros
			case "Monsters":
				monsterMutex.Lock()
				monsters = list.New()
				data := sData.Data.(map[string]interface{})
				for _, m := range data {
					monster := m.(map[string]interface{})
					x := monster["X"].(float64)
					y := monster["Y"].(float64)
					id := monster["Id"].(float64)
					mRight := monster["Movingright"].(bool)
					sBool := monster["Spritebool"].(bool)
					mType := monster["MonsterType"].(float64)
					nMonster := CreateMonster(x, y, sBool, mRight, mType, byte(id))
					monsters.PushFront(nMonster)
				}
				if monsters.Len() == 0 {
					won = true
				}
				monsterMutex.Unlock()
			//Recebe do servidor um tiro efetuado por um monstro
			case "MonsterShot":
				data := sData.Data.(map[string]interface{})
				mShotMutex.Lock()
				mShots.PushFront(CreateShot(data["X"].(float64), data["Y"].(float64), true))
				mShotMutex.Unlock()
			}
		}
	}()
	fps := time.Tick(time.Second / 480)
	for !win.Closed() {
		win.Clear(colornames.Black)
		if !won && !gameover {
			if !dead {
				if win.JustPressed(pixelgl.KeyRight) {
					if ship.X < winW-5 {
						ship.X += 5
						sendData(&bebShip, ship, "SpaceShip")
					}
				}
				if win.JustPressed(pixelgl.KeyLeft) {
					if ship.X > 5 {
						ship.X -= 5
						sendData(&bebShip, ship, "SpaceShip")
					}
				}
				if win.JustPressed(pixelgl.KeySpace) {
					sendData(&bebOthers, CreateShot(ship.X, ship.Y, false), "Shot")
				}
			}
			shipMutex.Lock()
			if len(shipDict) == 0 && gamestarted {
				gameover = true
			}
			for _, v := range shipDict {
				vd := v.(map[string]interface{})
				rship := CreateSpaceship()
				rship.X = vd["X"].(float64)
				rship.Y = vd["Y"].(float64)
				rship.Draw(win)
			}
			shipMutex.Unlock()

			//Verifica se um tiro atingiu um monstro
			monsterMutex.Lock()
			for m := monsters.Front(); m != nil; m = m.Next() {
				monster := m.Value.(*Monster)
				monster.Draw(win)
				shotMutex.Lock()
				for s := shots.Front(); s != nil; s = s.Next() {
					shot := s.Value.(*Shot)
					if shot.X >= monster.X-monster.Monstersprite1.Frame().W()/2 &&
						shot.X <= monster.X+monster.Monstersprite1.Frame().W()/2 &&
						shot.Y >= monster.Y-monster.Monstersprite1.Frame().H()/2 &&
						shot.Y <= monster.Y+monster.Monstersprite1.Frame().H()/2 {
						sendData(&bebOthers, monster.Id, "KillMonster")
						monsters.Remove(m)
						shots.Remove(s)
					}
				}
				shotMutex.Unlock()
			}
			monsterMutex.Unlock()

			//Verifica se um tiro atingiu uma nave
			mShotMutex.Lock()
			for s := mShots.Front(); s != nil; s = s.Next() {
				shot := s.Value.(*Shot)
				shipMutex.Lock()
				for k, v := range shipDict {
					vd := v.(map[string]interface{})
					rship := CreateSpaceship()
					rship.X = vd["X"].(float64)
					rship.Y = vd["Y"].(float64)
					if shot.X >= rship.X-rship.Shipsprite.Frame().W()/2 &&
						shot.X <= rship.X+rship.Shipsprite.Frame().W()/2 &&
						shot.Y >= rship.Y-rship.Shipsprite.Frame().H()/2 &&
						shot.Y <= rship.Y+rship.Shipsprite.Frame().H()/2 {
						mShots.Remove(s)
						delete(shipDict, k)
						if rship.X == ship.X && rship.Y == ship.Y {
							sendData(&bebOthers, nil, "KillShip")
							dead = true
						}
					}
				}
				shipMutex.Unlock()
			}
			mShotMutex.Unlock()

			shotMutex.Lock()
			for s := shots.Front(); s != nil; s = s.Next() {
				shot := s.Value.(*Shot)
				if shot.Y > winH {
					shots.Remove(s)
				} else {
					shot.Draw(win)
				}
			}
			shotMutex.Unlock()
			mShotMutex.Lock()
			for s := mShots.Front(); s != nil; s = s.Next() {
				shot := s.Value.(*Shot)
				if shot.Y < 0 {
					mShots.Remove(s)
				} else {
					shot.Draw(win)
				}
			}
			mShotMutex.Unlock()

		} else {
			if won {
				wonTxt.Draw(win, pixel.IM)
			}
			if gameover {
				gameoverTxt.Draw(win, pixel.IM)
			}
		}
		win.Update()
		<-fps
	}
	sendData(&bebOthers, nil, "CloseConnection")

}

func main() {
	pixelgl.Run(run)
}
