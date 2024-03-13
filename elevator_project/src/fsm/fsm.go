package fsm

import (
	"driver/config"
	"driver/elevator"
	"driver/elevator_io"
	"driver/requests"
	"fmt"
	"time"
)

/*
// One single function for the Final State Machine, to be run as a goroutine from main
func Fsm(
	id string,
	ch_arrivalFloor <-chan int,
	ch_localRequests <-chan [config.N_FLOORS][config.N_BUTTONS]bool,
	ch_doorObstruction <-chan bool,
	ch_stopButton <-chan bool,
	ch_completedRequests chan<- elevator_io.ButtonEvent,
	ch_elevatorStateToAssigner chan<- map[string]elevator.ElevatorState,
	ch_elevatorStateToNetwork chan<- elevator.ElevatorState,
	ch_FSMLifeLine chan<- int,
) {

	// Initializing
	fmt.Printf("*****INITIALIZING ELEVATOR*****\n")
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

	elevator.SendLocalElevatorState(id, localElevator, ch_elevatorStateToAssigner, ch_elevatorStateToNetwork)

	// "For-Select" to supervise the different channels/events that changes the FSM
	for {
		ch_FSMLifeLine <- 1
		select {
		case localRequests := <-ch_localRequests:
			fmt.Printf("Entered Local requests in FSM\n")

			localElevator.Requests = localRequests
			localElevator.Elevator_print()

			switch localElevator.Behaviour {
			case elevator.EB_DoorOpen:
				fmt.Println("In first door open case, BEFORE IF. Direction is:", elevator.ElevDirnToString(localElevator.Dirn), elevator.ElevBehaviourToString(localElevator.Behaviour))

				elevator_io.SetDoorOpenLamp(true)
				doorTimer.Reset(time.Duration(config.DoorOpenDurationSec) * time.Second)

				if requests.Here(&localElevator) && (localElevator.Dirn == elevator_io.MD_Stop) {
					fmt.Println("In first door open case, AFTER IF. Direction is:", elevator.ElevDirnToString(localElevator.Dirn), elevator.ElevBehaviourToString(localElevator.Behaviour))
					requests.ClearAtCurrentFloor(&localElevator, ch_completedRequests)
				}
				localElevator.HoldDoorOpenIfObstruction(&prevObstruction, doorTimer, ch_doorObstruction)
				fmt.Println("In first door open case, END of the case. Direction is:", elevator.ElevDirnToString(localElevator.Dirn), elevator.ElevBehaviourToString(localElevator.Behaviour))

			case elevator.EB_Idle:
				requests.ChooseDirnAndBehaviour(&localElevator)
				fmt.Println("In idle case. Direction is:", elevator.ElevDirnToString(localElevator.Dirn), elevator.ElevBehaviourToString(localElevator.Behaviour))

				switch localElevator.Behaviour {
				case elevator.EB_Moving:
					elevator_io.SetMotorDirection(localElevator.Dirn)
					fmt.Println("In Moving-case. Direction is:", elevator.ElevDirnToString(localElevator.Dirn), elevator.ElevBehaviourToString(localElevator.Behaviour))

				case elevator.EB_DoorOpen:
					fmt.Println("In second door open case. Direction is:", elevator.ElevDirnToString(localElevator.Dirn), elevator.ElevBehaviourToString(localElevator.Behaviour))
					elevator_io.SetDoorOpenLamp(true)
					doorTimer.Reset(time.Duration(config.DoorOpenDurationSec) * time.Second)
					requests.ClearAtCurrentFloor(&localElevator, ch_completedRequests)
					localElevator.HoldDoorOpenIfObstruction(&prevObstruction, doorTimer, ch_doorObstruction)
				}
			} //switch e.behaviour

		case newFloor := <-ch_arrivalFloor:
			fmt.Printf("Entered new floor in FSM\n")
			localElevator.Elevator_print()

			localElevator.Floor = newFloor
			elevator_io.SetFloorIndicator(localElevator.Floor)

			switch localElevator.Behaviour {
			case elevator.EB_Moving:
				if requests.ShouldStop(&localElevator) {
					elevator_io.SetMotorDirection(elevator_io.MD_Stop)
					elevator_io.SetDoorOpenLamp(true)
					doorTimer.Reset(time.Duration(config.DoorOpenDurationSec) * time.Second)
					requests.ClearAtCurrentFloor(&localElevator, ch_completedRequests)
					localElevator.HoldDoorOpenIfObstruction(&prevObstruction, doorTimer, ch_doorObstruction)
					localElevator.Behaviour = elevator.EB_DoorOpen
				}
			}

		// This channel automatically "transmits" when the timer times out.
		case <-doorTimer.C:
			fmt.Printf("Entered doorTimeout in FSM\n")
			localElevator.Elevator_print()

			switch localElevator.Behaviour {
			case elevator.EB_DoorOpen:
				if requests.Here(&localElevator) {
					requests.AnnounceDirectionChange(&localElevator)
					requests.ClearAtCurrentFloor(&localElevator, ch_completedRequests)
					time.Sleep(time.Duration(config.DoorOpenDurationSec) * time.Second)
				}
				localElevator.HoldDoorOpenIfObstruction(&prevObstruction, doorTimer, ch_doorObstruction)
				elevator_io.SetDoorOpenLamp(false)
				requests.ChooseDirnAndBehaviour(&localElevator)

				switch localElevator.Behaviour {
				case elevator.EB_Moving:
					elevator_io.SetMotorDirection(localElevator.Dirn)
				}
			}

		case obstruction := <-ch_doorObstruction:
			fmt.Printf("Entered obstruction in FSM\n")
			prevObstruction = obstruction

		case <-ch_stopButton:
			fmt.Printf("Entered Stop Button in FSM\n")
			localElevator.Elevator_print()

			switch localElevator.Behaviour {
			case elevator.EB_DoorOpen:
				elevator_io.SetDoorOpenLamp(true)
				doorTimer.Reset(time.Duration(config.DoorOpenDurationSec) * time.Second)
				localElevator.HoldDoorOpenIfObstruction(&prevObstruction, doorTimer, ch_doorObstruction)

			case elevator.EB_Moving:
				elevator_io.SetMotorDirection(elevator_io.MD_Stop)
			}

			localElevator.StallWhileStopButtonActive(ch_stopButton)

			// Makes sure the elevator keeps going after when stopButton is no longer active.
			switch localElevator.Behaviour {
			case elevator.EB_Moving:
				elevator_io.SetMotorDirection(localElevator.Dirn)

			case elevator.EB_DoorOpen:
				doorTimer.Reset(time.Duration(config.DoorOpenDurationSec) * time.Second)
				requests.ClearAtCurrentFloor(&localElevator, ch_completedRequests)
				localElevator.HoldDoorOpenIfObstruction(&prevObstruction, doorTimer, ch_doorObstruction)
			}
			localElevator.Elevator_print()

		default:
			// NOP
		} //select

		if prevLocalElevator != localElevator {
			prevLocalElevator = localElevator
			elevator.SendLocalElevatorState(id, localElevator, ch_elevatorStateToAssigner, ch_elevatorStateToNetwork)
			localElevator.Elevator_print()
		}
	} //For
} //Fsm
*/

// One single function for the Final State Machine, to be run as a goroutine from main
func Fsm(
	id string,
	ch_arrivalFloor <-chan int,
	ch_localRequests <-chan [config.N_FLOORS][config.N_BUTTONS]bool,
	ch_doorObstruction <-chan bool,
	ch_stopButton <-chan bool,
	ch_completedRequests chan<- elevator_io.ButtonEvent,
	ch_elevatorStateToAssigner chan<- map[string]elevator.ElevatorState,
	ch_elevatorStateToNetwork chan<- elevator.ElevatorState,
	ch_FSMLifeLine chan<- int,
) {

	// Initializing
	fmt.Printf("*****INITIALIZING ELEVATOR*****\n")
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

	elevator.SendLocalElevatorState(id, localElevator, ch_elevatorStateToAssigner, ch_elevatorStateToNetwork)

	// "For-Select" to supervise the different channels/events that changes the FSM
	for {
		select {
		case localRequests := <-ch_localRequests:
			fmt.Printf("Entered Local requests in FSM\n")

			localElevator.Requests = localRequests
			localElevator.Elevator_print()

			switch localElevator.Behaviour {
			case elevator.EB_Moving:
				//NOP
			case elevator.EB_DoorOpen:
				if requests.Here(&localElevator) && (localElevator.Dirn == elevator_io.MD_Stop) {
					requests.ClearAtCurrentFloor(&localElevator, ch_completedRequests)
					doorTimer.Reset(time.Duration(config.DoorOpenDurationSec) * time.Second)
				}
				//localElevator.HoldDoorOpenIfObstruction(&prevObstruction, doorTimer, ch_doorObstruction)

			case elevator.EB_Idle:
				if requests.Here(&localElevator) && (localElevator.Dirn == elevator_io.MD_Stop) {
					requests.ClearAtCurrentFloor(&localElevator, ch_completedRequests)
					elevator_io.SetDoorOpenLamp(true)
					doorTimer.Reset(time.Duration(config.DoorOpenDurationSec) * time.Second)
					localElevator.Behaviour = elevator.EB_DoorOpen

					//localElevator.HoldDoorOpenIfObstruction(&prevObstruction, doorTimer, ch_doorObstruction)
				} else {
					// See if requests are elsewhere
					requests.ChooseDirnAndBehaviour(&localElevator)
					if localElevator.Behaviour == elevator.EB_Moving {
						elevator_io.SetMotorDirection(localElevator.Dirn)
					}
				}

			} //switch e.behaviour

		case newFloor := <-ch_arrivalFloor:
			fmt.Printf("Entered new floor in FSM\n")
			localElevator.Elevator_print()

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

					//localElevator.HoldDoorOpenIfObstruction(&prevObstruction, doorTimer, ch_doorObstruction)
				}
			case elevator.EB_DoorOpen, elevator.EB_Idle:
				//NOP
			}

		// This channel automatically "transmits" when the timer times out.
		case <-doorTimer.C:
			fmt.Printf("Entered doorTimeout in FSM\n")
			localElevator.Elevator_print()

			switch localElevator.Behaviour {
			case elevator.EB_Idle, elevator.EB_Moving:
				//NOP
			case elevator.EB_DoorOpen:
				// This "if" happens when hallButton in direction was cleared at new floor,
				// 	but should wait longer if hallButton in other direction should also be cleared:
				if requests.Here(&localElevator) {
					requests.AnnounceDirectionChange(&localElevator)
					requests.ClearAtCurrentFloor(&localElevator, ch_completedRequests)
					time.Sleep(time.Duration(config.DoorOpenDurationSec) * time.Second)
				}
				//localElevator.HoldDoorOpenIfObstruction(&prevObstruction, doorTimer, ch_doorObstruction)
				if prevObstruction {
					fmt.Println("Door is obstructed")
					prevObstruction = <-ch_doorObstruction
					time.Sleep(time.Duration(config.DoorOpenDurationSec) * time.Second)
				}
				elevator_io.SetDoorOpenLamp(false)

				//Decides further action:
				requests.ChooseDirnAndBehaviour(&localElevator)
				if localElevator.Behaviour == elevator.EB_Moving {
					elevator_io.SetMotorDirection(localElevator.Dirn)
				}
			}

		case obstruction := <-ch_doorObstruction:
			fmt.Printf("Entered obstruction in FSM\n")
			prevObstruction = obstruction

		case <-ch_stopButton:
			fmt.Printf("Entered Stop Button in FSM\n")
			localElevator.Elevator_print()

			if localElevator.Behaviour == elevator.EB_Moving {
				elevator_io.SetMotorDirection(elevator_io.MD_Stop)
			}

			localElevator.StallWhileStopButtonActive(ch_stopButton)

			// Makes sure the elevator keeps going when stopButton is no longer active.
			switch localElevator.Behaviour {
			case elevator.EB_Moving:
				elevator_io.SetMotorDirection(localElevator.Dirn)

			case elevator.EB_DoorOpen:
				doorTimer.Reset(time.Duration(config.DoorOpenDurationSec) * time.Second)
				localElevator.HoldDoorOpenIfObstruction(&prevObstruction, doorTimer, ch_doorObstruction)
			}
			localElevator.Elevator_print()

		default:
			// NOP
		} //select

		if prevLocalElevator != localElevator {
			prevLocalElevator = localElevator
			elevator.SendLocalElevatorState(id, localElevator, ch_elevatorStateToAssigner, ch_elevatorStateToNetwork)
			localElevator.Elevator_print()
		}
	} //For
} //Fsm
