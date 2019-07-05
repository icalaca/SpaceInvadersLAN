package main

import (
	"bytes"
	"container/list"
	"encoding/json"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"

	BEBcast "../BEBcast"
	"../Structs"
)

var clientList = list.New()
var winW = 400.0
var winH = 300.0
var mutex = &sync.Mutex{}
var monsterMutex = &sync.Mutex{}
var shipMutex = &sync.Mutex{}
var nMonsters = 11
var spacingX = 30.0

//Funcao para auxiliar no envio dos dados
func sendData(beb *BEBcast.BestEffortBroadcast_Module, ip string, port int, data interface{}, datatype string) {
	serverAddr := []string{ip + ":" + strconv.Itoa(port)}
	sData := Structs.ServerData{datatype, data}
	req := BEBcast.BestEffortBroadcast_Req_Message{
		Addresses: serverAddr,
		Message:   sData}
	beb.Req <- req
}

func CreateMonster(monstertype int, id byte) *Structs.Monster {
	monster := Structs.Monster{}
	monster.Id = id
	monster.X = 0
	monster.Y = 0
	monster.MonsterType = monstertype
	monster.Spritebool = true
	monster.Movingright = true
	return &monster
}

//Inicializa os monstros seguindo a formacao classica do Space Invaders
func initMonsters(monsters map[int]*Structs.Monster) {
	id := 0
	spacingY := 30.0
	initX := (winW / 2) - ((spacingX * float64(nMonsters)) / 2)
	x := initX
	y := winH - 20
	mrep := 1
	for i := 0; i < 3; i++ {
		if i != 0 {
			mrep = 2
		}
		for k := 0; k < mrep; k++ {
			for j := 0; j < nMonsters; j++ {
				monster := CreateMonster((i % 3), byte(id))
				monster.X = x
				monster.Y = y
				monsters[id] = monster
				id++
				x += spacingX
			}
			x = initX
			y -= spacingY
		}
	}
}

//Movimenta os monstros e alterna entre os sprites
func dance(monsters map[int]*Structs.Monster) {
	minX := math.MaxFloat64
	maxX := math.SmallestNonzeroFloat64
	minY := math.MaxFloat64
	initX := (winW / 2) - ((spacingX * float64(nMonsters)) / 2)
	for _, m := range monsters {
		monsterX := m.X
		monsterY := m.Y
		if monsterX < minX {
			minX = monsterX
		}
		if monsterX > maxX {
			maxX = monsterX
		}
		if monsterY < minY {
			minY = monsterY
		}
	}
	for _, m := range monsters {
		monster := m
		if monster.Movingright {
			monster.X += 5
			if maxX > winW-initX {
				monster.Movingright = false
				if minY > winH/10 {
					monster.Y -= 10
				}
			}
		} else {
			monster.X -= 5
			if minX < initX {
				monster.Movingright = true
				if minY > winH/10 {
					monster.Y -= 10
				}
			}
		}
		if monster.Spritebool {
			monster.Spritebool = false
		} else {
			monster.Spritebool = true
		}

	}

}

func main() {
	monsters := make(map[int]*Structs.Monster)
	shipDict := make(map[string]interface{})
	//shipDict := make(map[string]Structs.Spaceship)

	bebMonsters := BEBcast.BestEffortBroadcast_Module{
		Req: make(chan BEBcast.BestEffortBroadcast_Req_Message),
		Ind: make(chan BEBcast.BestEffortBroadcast_Ind_Message)}
	bebShip := BEBcast.BestEffortBroadcast_Module{
		Req: make(chan BEBcast.BestEffortBroadcast_Req_Message),
		Ind: make(chan BEBcast.BestEffortBroadcast_Ind_Message)}
	bebOthers := BEBcast.BestEffortBroadcast_Module{
		Req: make(chan BEBcast.BestEffortBroadcast_Req_Message),
		Ind: make(chan BEBcast.BestEffortBroadcast_Ind_Message)}

	bebMonsters.Init("127.0.0.1:4001", 8192)
	bebShip.Init("127.0.0.1:4002", 1024)
	bebOthers.Init("127.0.0.1:4003", 1024)

	initMonsters(monsters)

	go func() {
		var monster *Structs.Monster
		var mShot Structs.Shot
		mTick := 0.0
		for {
			if mTick > 8000 {
				monsterMutex.Lock()
				if len(monsters) > 0 {
					dance(monsters)
					//Seleciona um monstro aleatorio para atirar
					r := rand.Intn(len(monsters))
					for _, m := range monsters {
						if r == 0 {
							monster = m
						}
						r--
					}
					mShot = Structs.Shot{nil, true, monster.X, monster.Y}
				}
				mutex.Lock()
				//Envia a lista de monstros pros clientes e um tiro
				//efetuado por um monstro aleatorio
				for c := clientList.Front(); c != nil; c = c.Next() {
					client := c.Value.(Structs.Client)
					sendData(&bebMonsters, client.IP, client.Port, monsters, "Monsters")
					if len(monsters) > 0 {
						sendData(&bebMonsters, client.IP, client.Port, mShot, "MonsterShot")
					}
				}
				mutex.Unlock()
				monsterMutex.Unlock()
				mTick = 0
			} else {
				mTick += 0.00001
			}
		}
	}()

	go func() {
		for {
			in := <-bebShip.Ind
			cData := Structs.ClientData{}
			reqBodyBytes := new(bytes.Buffer)
			json.NewDecoder(reqBodyBytes).Decode(in.Message)
			json.Unmarshal([]byte(in.Message.(string)), &cData)
			clientIP := in.From[:strings.IndexByte(in.From, ':')]
			switch cData.Datatype {
			//Altera o estado da nave do cliente e espalha o estado
			case "SpaceShip":
				data := cData.Data.(map[string]interface{})
				key := clientIP + ":" + cData.ClientPortBind
				//ship := Structs.Spaceship{nil, data["X"].(float64), data["Y"].(float64)}
				shipMutex.Lock()
				shipDict[key] = data
				mutex.Lock()
				for c := clientList.Front(); c != nil; c = c.Next() {
					client := c.Value.(Structs.Client)
					sendData(&bebShip, client.IP, client.Port+1, shipDict, "ShipDict")
				}
				mutex.Unlock()
				shipMutex.Unlock()
			}
		}
	}()

	go func() {
		for {
			in := <-bebOthers.Ind
			cData := Structs.ClientData{}
			reqBodyBytes := new(bytes.Buffer)
			json.NewDecoder(reqBodyBytes).Decode(in.Message)
			json.Unmarshal([]byte(in.Message.(string)), &cData)
			clientIP := in.From[:strings.IndexByte(in.From, ':')]
			clientPort, _ := strconv.Atoi(cData.ClientPortBind)

			switch cData.Datatype {
			//Recebe os dados do cliente e guarda na lista de clientes
			case "PortHandShake":
				clientList.PushFront(Structs.Client{clientIP, clientPort})
			//Espalha o tiro efetuado para os clientes
			case "Shot":
				data := cData.Data.(map[string]interface{})
				// shot := Structs.Shot{nil, false, data["X"].(float64), data["Y"].(float64)}
				mutex.Lock()
				for c := clientList.Front(); c != nil; c = c.Next() {
					client := c.Value.(Structs.Client)
					sendData(&bebOthers, client.IP, client.Port+2, data, "ShotDict")
				}
				mutex.Unlock()
				//Recebe do cliente a informacao de que um monstro foi morto
				//e remove o monstro da lista
			case "KillMonster":
				data := cData.Data.(float64)
				monsterMutex.Lock()
				delete(monsters, int(data))
				monsterMutex.Unlock()
			//Recebe do cliente a informacao de que a nave foi morta
			//e remove a nave da lista
			case "KillShip":
				shipMutex.Lock()
				delete(shipDict, clientIP+":"+cData.ClientPortBind)
				shipMutex.Unlock()
			//Remove o cliente da lista de clientes, a nave da lista de naves
			//e espalha as alteracoes
			case "CloseConnection":
				mutex.Lock()
				for c := clientList.Front(); c != nil; c = c.Next() {
					client := c.Value.(Structs.Client)
					if client.IP == clientIP && client.Port == clientPort {
						clientList.Remove(c)
						break
					}
				}
				mutex.Unlock()

				shipMutex.Lock()
				_, exists := shipDict[clientIP+":"+cData.ClientPortBind]
				if exists {
					delete(shipDict, clientIP+":"+cData.ClientPortBind)
				}
				shipMutex.Unlock()

				mutex.Lock()
				for c := clientList.Front(); c != nil; c = c.Next() {
					client := c.Value.(Structs.Client)
					sendData(&bebShip, client.IP, client.Port+1, shipDict, "ShipDict")
				}
				mutex.Unlock()
			}
		}
	}()

	blq := make(chan int)
	<-blq
}
