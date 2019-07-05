// Construido como parte da disciplina de Sistemas Distribuidos
// Semestre 2018/2  -  PUCRS - Escola Politecnica
// Estudantes:  Andre Antonitsch e Rafael Copstein
// Professor: Fernando Dotti  (www.inf.pucrs.br/~fldotti)
// Algoritmo baseado no livro:
// Introduction to Reliable and Secure Distributed Programming
// Christian Cachin, Rachid Gerraoui, Luis Rodrigues

//package BestEffortBroadcast

package BEBcast

import "fmt"

import PP2PLink "../PP2PLink"

type BestEffortBroadcast_Req_Message struct {
	Addresses []string
	Message   interface{}
}

type BestEffortBroadcast_Ind_Message struct {
	From    string
	Message interface{}
}

type BestEffortBroadcast_Module struct {
	Ind      chan BestEffortBroadcast_Ind_Message
	Req      chan BestEffortBroadcast_Req_Message
	Pp2plink PP2PLink.PP2PLink
}

func (module BestEffortBroadcast_Module) Init(address string, bsize int) {

	fmt.Println("Init BEB!")
	module.Pp2plink = PP2PLink.PP2PLink{
		Req:     make(chan PP2PLink.PP2PLink_Req_Message),
		Ind:     make(chan PP2PLink.PP2PLink_Ind_Message),
		Bufsize: bsize}
	/*if !bbuf {
		module.Pp2plink.Init(address, 8192)
	} else {
		module.Pp2plink.Init(address, 1024)
	}*/
	module.Pp2plink.Init(address)
	module.Start()

}

func (module BestEffortBroadcast_Module) Start() {

	go func() {
		for {
			select {
			case y := <-module.Req:
				module.Broadcast(y)
			case y := <-module.Pp2plink.Ind:
				module.Deliver(PP2PLink2BEB(y))
			}
		}
	}()

}

func (module BestEffortBroadcast_Module) Broadcast(message BestEffortBroadcast_Req_Message) {

	for i := 0; i < len(message.Addresses); i++ {
		msg := BEB2PP2PLink(message)
		msg.To = message.Addresses[i]
		module.Pp2plink.Req <- msg
		//fmt.Println("Sent to " + message.Addresses[i])
		//fmt.Println(module.Pp2plink.Bufsize)
	}

}

func (module BestEffortBroadcast_Module) Deliver(message BestEffortBroadcast_Ind_Message) {

	//fmt.Println("Received msg from " + message.From)
	module.Ind <- message
	//fmt.Println("# End BEB Received")

}

func BEB2PP2PLink(message BestEffortBroadcast_Req_Message) PP2PLink.PP2PLink_Req_Message {

	return PP2PLink.PP2PLink_Req_Message{
		To:      message.Addresses[0],
		Message: message.Message}

}

func PP2PLink2BEB(message PP2PLink.PP2PLink_Ind_Message) BestEffortBroadcast_Ind_Message {

	return BestEffortBroadcast_Ind_Message{
		From:    message.From,
		Message: message.Message}

}
