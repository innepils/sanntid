package cost

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"src/config"
	"src/elevator"
)

type HRAInput struct {
	HallRequests      [config.N_FLOORS][config.N_BUTTONS - 1]bool 	`json:"hallRequests"`
	StatesofElevators map[string]elevator.HRAElevatorState          `json:"states"`
}

func Cost(
	nodeID 				string,
	hallRequests 		[config.N_FLOORS][config.N_BUTTONS - 1]bool,
	localElevator 		map[string]elevator.HRAElevatorState,
	externalElevators 	[]byte,
	) [][config.N_BUTTONS-1]bool {

	input := HRAInput{
		HallRequests: hallRequests,
		StatesofElevators: map[string]elevator.HRAElevatorState{
			nodeID: localElevator[nodeID],
		},
	}

	var externalElevatorsDecoded map[string]elevator.HRAElevatorState
	json.Unmarshal(externalElevators, &externalElevatorsDecoded)
	for key, value := range externalElevatorsDecoded {
		input.StatesofElevators[key] = value

	}

	jsonBytes, err := json.Marshal(input)
	if err != nil {
		fmt.Println("json.Marshal error: ", err)
	}

	ret, err := exec.Command("./hall_request_assigner/hall_request_assigner", "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		fmt.Println("exec.Command error: ", err)
		fmt.Println(string(ret))
	}

	output := new(map[string][][config.N_BUTTONS-1]bool)
	err = json.Unmarshal(ret, &output)
	if err != nil {
		fmt.Println("json.Unmarshal error: ", err)
	}

	return (*output)[nodeID]
}
