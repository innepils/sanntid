package fsm

import (
	"driver/config"
	"driver/elevator"
	"driver/elevator_io"
	"driver/requests"
	"time"
)

func FSM(
	nodeID 						string,
	ch_localRequests 			<-chan [config.N_FLOORS][config.N_BUTTONS]bool,
	ch_arrivalFloor 			<-chan int,
	ch_doorObstruction 			<-chan bool,
	ch_stopButton				<-chan bool,
	ch_completedRequests 		chan<- elevator_io.ButtonEvent,
	ch_elevatorStateToAssigner 	chan<- map[string]elevator.ElevatorState,
	ch_elevatorStateToNetwork 	chan<- elevator.ElevatorState,
	ch_FSMDeadlock 				chan<- int,
) {

	// Initilalize variables
	localElevator := elevator.UninitializedElevator()
	prevLocalElevator := localElevator
	prevObstruction := false

	// If elevator is between floors, run it downwards until a floor is reached.
	elevator_io.SetMotorDirection(elevator_io.MD_Down)
	newFloor := <-ch_arrivalFloor
	elevator_io.SetMotorDirection(elevator_io.MD_Stop)
	localElevator.Floor = newFloor
	elevator_io.SetFloorIndicator(localElevator.Floor)

	// Initialize door
	elevator_io.SetDoorOpenLamp(false)
	doorTimer := time.NewTimer(time.Duration(config.DoorOpenDurationSec) * time.Second)

	elevator.SendLocalElevatorState(nodeID, localElevator, ch_elevatorStateToAssigner, ch_elevatorStateToNetwork)

	// "For-Select" to supervise the different channels/events
	for {
		ch_FSMDeadlock <- 1
		select {
		case localRequests := <-ch_localRequests:
			localElevator.Requests = localRequests

			switch localElevator.Behaviour {
			case elevator.EB_Moving:
				//NOP
			case elevator.EB_DoorOpen:
				if requests.Here(&localElevator) && (localElevator.Dirn == elevator_io.MD_Stop) {
					requests.ClearAtCurrentFloor(&localElevator, ch_completedRequests)
					doorTimer.Reset(time.Duration(config.DoorOpenDurationSec) * time.Second)
				}
			case elevator.EB_Idle:
				if requests.Here(&localElevator) && (localElevator.Dirn == elevator_io.MD_Stop) {
					requests.ClearAtCurrentFloor(&localElevator, ch_completedRequests)
					elevator_io.SetDoorOpenLamp(true)
					doorTimer.Reset(time.Duration(config.DoorOpenDurationSec) * time.Second)
					localElevator.Behaviour = elevator.EB_DoorOpen
				} else {
					requests.ChooseDirnAndBehaviour(&localElevator)
					if localElevator.Behaviour == elevator.EB_Moving {
						elevator_io.SetMotorDirection(localElevator.Dirn)
					}
				}
			}

		case newFloor := <-ch_arrivalFloor:
			localElevator.Floor = newFloor
			elevator_io.SetFloorIndicator(localElevator.Floor)

			switch localElevator.Behaviour {
			case elevator.EB_Moving:
				if requests.ShouldStop(&localElevator) {
					elevator_io.SetMotorDirection(elevator_io.MD_Stop)
					requests.ClearAtCurrentFloor(&localElevator, ch_completedRequests)
					elevator_io.SetDoorOpenLamp(true)
					doorTimer.Reset(time.Duration(config.DoorOpenDurationSec) * time.Second)
					localElevator.Behaviour = elevator.EB_DoorOpen
				}
			case elevator.EB_DoorOpen, elevator.EB_Idle:
				//NOP
			} 

		// This channel "transmits" when door timeouts.
		case <-doorTimer.C:
			switch localElevator.Behaviour {
			case elevator.EB_Idle, elevator.EB_Moving:
				//NOP
			case elevator.EB_DoorOpen:
				if requests.Here(&localElevator) {
					requests.AnnounceDirectionChange(&localElevator)
					requests.ClearAtCurrentFloor(&localElevator, ch_completedRequests)
					time.Sleep(time.Duration(config.DoorOpenDurationSec) * time.Second)
				}
				// Keeps the door open while obstruction is active
				for prevObstruction{
					ch_FSMDeadlock <- 1
					select {
					case prevObstruction = <-ch_doorObstruction:
							time.Sleep(time.Duration(config.DoorOpenDurationSec) * time.Second)
					default:
						//NOP
						}
				}
				elevator_io.SetDoorOpenLamp(false)

				requests.ChooseDirnAndBehaviour(&localElevator)
				if localElevator.Behaviour == elevator.EB_Moving {
					elevator_io.SetMotorDirection(localElevator.Dirn)
				}
			} //switch localElevator.behaviour

		case obstruction := <-ch_doorObstruction:
			prevObstruction = obstruction

		case <-ch_stopButton:
			if localElevator.Behaviour == elevator.EB_Moving {
				elevator_io.SetMotorDirection(elevator_io.MD_Stop)
			}
			// Keeps the elevator and FSM stalled while stopButton is pressed
			stopButtonPressed := true
			for stopButtonPressed {
				ch_FSMDeadlock <- 1
				stopButtonPressed = false
				stopButtonPressed = <-ch_stopButton
			}
			switch localElevator.Behaviour {
			case elevator.EB_Moving:
				elevator_io.SetMotorDirection(localElevator.Dirn)

			case elevator.EB_DoorOpen:
				doorTimer.Reset(time.Duration(config.DoorOpenDurationSec) * time.Second)
			} //switch localElevator.behaviour
		default:
			// NOP
		} //select

		if prevLocalElevator != localElevator {
			prevLocalElevator = localElevator
			elevator.SendLocalElevatorState(nodeID, localElevator, ch_elevatorStateToAssigner, ch_elevatorStateToNetwork)
		}
	} //For
} //FSM
