package main

import (
	"net"

	"__NAME__/crypto"
	"__NAME__/protocol"
)

func recvMessage(conn net.Conn, sessionKey []byte) (protocol.Message, error) {
	var msg protocol.Message
	data, err := protocol.RecvMsg(conn)
	if err != nil {
		return msg, err
	}
	plaintext, err := crypto.DecryptData(data, sessionKey)
	if err != nil {
		return msg, err
	}
	if err := protocol.Unmarshal(plaintext, &msg); err != nil {
		return msg, err
	}
	return msg, nil
}

func sendMessage(conn net.Conn, sessionKey []byte, msgType int8, objects [][]byte) error {
	payload, err := protocol.Marshal(protocol.Message{Type: msgType, Object: objects})
	if err != nil {
		return err
	}
	enc, err := crypto.EncryptData(payload, sessionKey)
	if err != nil {
		return err
	}
	return protocol.SendMsg(conn, enc)
}
