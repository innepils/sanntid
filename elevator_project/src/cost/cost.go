package cost

import (
	"driver/config"
	"driver/elevator"
	"encoding/json"
	"fmt"
	"os/exec"
)

type HRAInput struct {
	HallRequests      [config.N_FLOORS][config.N_BUTTONS - 1]bool `json:"hallRequests"`
	StatesofElevators map[string]elevator.ElevatorState           `json:"states"`
}

func Cost(
	id string,
	hallRequests [config.N_FLOORS][config.N_BUTTONS - 1]bool,
	localElevator map[string]elevator.ElevatorState,
	externalElevators []byte) [][2]bool {

	input := HRAInput{
		HallRequests: hallRequests,
		StatesofElevators: map[string]elevator.ElevatorState{
			id: localElevator[id],
		},
	}

	var externalElevatorsDecoded map[string]elevator.ElevatorState
	json.Unmarshal(externalElevators, &externalElevatorsDecoded)
	//fmt.Println("external elevators: ", externalElevatorsDecoded)
	for key, value := range externalElevatorsDecoded {
		input.StatesofElevators[key] = value

	}
	//fmt.Println("Input elevators: ", input)

	jsonBytes, err := json.Marshal(input)
	if err != nil {
		fmt.Println("json.Marshal error: ", err)
		//die?
	}

	ret, err := exec.Command("./hall_request_assigner/hall_request_assigner", "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		fmt.Println("exec.Command error: ", err)
		fmt.Println(string(ret))
		//die?
	}

	output := new(map[string][][2]bool)
	err = json.Unmarshal(ret, &output)
	if err != nil {
		fmt.Println("json.Unmarshal error: ", err)
		//die?
	}

	//fmt.Println("Output of cost function:", output)
	return (*output)[id]
}
