package connection

// func DecodeID(data []byte) (s string, err error) {
// 	defer func() {
// 		if errx := recover(); err != nil {
// 			err = fmt.Errorf("%v", errx)
// 		}
// 	}()

// 	cursor := 0
// 	idLength := int(data[cursor])
// 	cursor += 1
// 	id := string(data[cursor : cursor+idLength])

// 	return id, nil
// }

// func EncodeID(id string) ([]byte, error) {
// 	data := []byte{}
// 	idBytes := []byte(id)
// 	idLength := len(idBytes)
// 	data = append(data, byte(idLength))
// 	data = append(data, idBytes...)

// 	return data, nil
// }
