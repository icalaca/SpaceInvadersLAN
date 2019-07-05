// Construido como parte da disciplina de Sistemas Distribuidos
// Semestre 2018/2  -  PUCRS - Escola Politecnica
// Estudantes:  Andre Antonitsch e Rafael Copstein
// Professor: Fernando Dotti  (www.inf.pucrs.br/~fldotti)
// Algoritmo baseado no livro:
// Introduction to Reliable and Secure Distributed Programming
// Christian Cachin, Rachid Gerraoui, Luis Rodrigues

package PP2PLink

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
)

type PP2PLink_Req_Message struct {
	To      string
	Message interface{}
}

type PP2PLink_Ind_Message struct {
	From    string
	Message interface{}
}

type PP2PLink struct {
	Ind     chan PP2PLink_Ind_Message
	Req     chan PP2PLink_Req_Message
	Bufsize int
	Run     bool
}

func (module PP2PLink) Init(address string) {

	fmt.Println("Init PP2PLink!")
	if module.Run {
		return
	}

	module.Run = true
	//module.Bufsize = bsize
	//module.bigbuf = true
	module.Start(address)
}
func Check(address string) bool {
	listen, lerr := net.Listen("tcp4", address)
	if lerr != nil {
		return false
	}
	listen.Close()
	return true
}
func (module PP2PLink) Start(address string) {

	go func() {
		var buf = make([]byte, module.Bufsize)
		listen, _ := net.Listen("tcp4", address)

		for {

			conn, err := listen.Accept()
			if err != nil {
				continue
			}
			len, _ := conn.Read(buf)
			conn.Close()
			content := make([]byte, len)
			copy(content, buf)

			msg := PP2PLink_Ind_Message{
				From:    conn.RemoteAddr().String(),
				Message: string(content)}

			module.Ind <- msg

		}
	}()

	go func() {
		for {
			message := <-module.Req
			module.Send(message)
		}
	}()

}

func (module PP2PLink) Send(message PP2PLink_Req_Message) {

	conn, err := net.Dial("tcp", message.To)
	if err != nil {
		fmt.Println(err)
		return
	}
	reqBodyBytes := new(bytes.Buffer)
	json.NewEncoder(reqBodyBytes).Encode(message.Message)
	conn.Write(reqBodyBytes.Bytes())
	//fmt.Fprintf(conn, reqBodyBytes.)
	conn.Close()

}
