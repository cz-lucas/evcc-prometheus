package evccprometheus

import "sync"

type Field[T any] struct {
	Value T
	Set   bool
	Null  bool
}

type BatteryState struct {
	Name Field[string]
	SOC  Field[float64]
}
type PVState struct {
	Name   Field[string]
	Power  Field[float64]
	Energy Field[float64]
}
type ConsumerState struct {
	Name   Field[string]
	Power  Field[float64]
	Energy Field[float64]
}
type GridState struct {
	Name     Field[string]
	Power    Field[float64]
	Energy   Field[float64]
	Currents Field[[]float64]
}
type ChargePointState struct {
	Name            Field[string]
	Charging        Field[bool]
	Connected       Field[bool]
	Power           Field[float64]
	ChargedEnergy   Field[float64]
	VehicleName     Field[string]
	VehicleSOC      Field[float64]
	VehicleRange    Field[float64]
	VehicleOdometer Field[float64]
}

type StateUpdate struct {
	Batteries          map[string]BatteryState
	ReplaceBattery     bool
	PVSystems          map[string]PVState
	ReplacePV          bool
	Consumers          map[string]ConsumerState
	ReplaceConsumer    bool
	GridMeters         map[string]GridState
	ReplaceGrid        bool
	ChargePoints       map[string]ChargePointState
	RemoveChargePoints []string
}

type Snapshot struct {
	Batteries    map[string]BatteryState
	PVSystems    map[string]PVState
	Consumers    map[string]ConsumerState
	GridMeters   map[string]GridState
	ChargePoints map[string]ChargePointState
}

type StateReader interface{ Snapshot() Snapshot }

type StateStore struct {
	mu           sync.RWMutex
	batteries    map[string]BatteryState
	pvSystems    map[string]PVState
	consumers    map[string]ConsumerState
	gridMeters   map[string]GridState
	chargePoints map[string]ChargePointState
}

// NewStateStore creates an empty, concurrency-safe store for EVCC domain state.
func NewStateStore() *StateStore {
	return &StateStore{
		batteries: make(map[string]BatteryState), pvSystems: make(map[string]PVState),
		consumers: make(map[string]ConsumerState), gridMeters: make(map[string]GridState),
		chargePoints: make(map[string]ChargePointState),
	}
}

// Apply merges partial fields, reconciles complete lists, then removes explicitly deleted loadpoints.
func (s *StateStore) Apply(update StateUpdate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if update.ReplaceBattery {
		s.batteries = reconcile(s.batteries, update.Batteries, mergeBattery)
	} else {
		applyMap(s.batteries, update.Batteries, mergeBattery)
	}
	if update.ReplacePV {
		s.pvSystems = reconcile(s.pvSystems, update.PVSystems, mergePV)
	} else {
		applyMap(s.pvSystems, update.PVSystems, mergePV)
	}
	if update.ReplaceConsumer {
		s.consumers = reconcile(s.consumers, update.Consumers, mergeConsumer)
	} else {
		applyMap(s.consumers, update.Consumers, mergeConsumer)
	}
	if update.ReplaceGrid {
		s.gridMeters = reconcile(s.gridMeters, update.GridMeters, mergeGrid)
	} else {
		applyMap(s.gridMeters, update.GridMeters, mergeGrid)
	}
	applyMap(s.chargePoints, update.ChargePoints, mergeChargePoint)
	for _, id := range update.RemoveChargePoints {
		delete(s.chargePoints, id)
	}
}

// Snapshot returns detached maps and copies mutable slices for safe concurrent scrapes.
func (s *StateStore) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := Snapshot{
		Batteries: cloneMap(s.batteries), PVSystems: cloneMap(s.pvSystems),
		Consumers: cloneMap(s.consumers), GridMeters: cloneMap(s.gridMeters),
		ChargePoints: cloneMap(s.chargePoints),
	}
	for id, grid := range snapshot.GridMeters {
		grid.Currents.Value = cloneSlice(grid.Currents.Value)
		snapshot.GridMeters[id] = grid
	}
	return snapshot
}

// reconcile replaces a category with updated IDs while retaining fields for matching entities.
func reconcile[T any](current, updates map[string]T, merge func(T, T) T) map[string]T {
	next := make(map[string]T, len(updates))
	for id, update := range updates {
		if existing, ok := current[id]; ok {
			update = merge(existing, update)
		}
		next[id] = update
	}
	return next
}

// applyMap merges updated entities without removing IDs absent from this partial update.
func applyMap[T any](current, updates map[string]T, merge func(T, T) T) {
	for id, update := range updates {
		current[id] = merge(current[id], update)
	}
}

// cloneMap copies a map so snapshots cannot mutate the store's map structure.
func cloneMap[T any](source map[string]T) map[string]T {
	clone := make(map[string]T, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

// cloneSlice returns an independent copy of a slice stored in a state snapshot.
func cloneSlice[T any](source []T) []T { return append([]T(nil), source...) }

// mergeField uses an update only when it represents a present wire field.
func mergeField[T any](current, update Field[T]) Field[T] {
	if update.Set {
		return update
	}
	return current
}

// mergeBattery applies only the battery fields present in an update.
func mergeBattery(current, update BatteryState) BatteryState {
	current.SOC = mergeField(current.SOC, update.SOC)
	return current
}

// mergePV applies only the PV fields present in an update.
func mergePV(current, update PVState) PVState {
	current.Power = mergeField(current.Power, update.Power)
	current.Energy = mergeField(current.Energy, update.Energy)
	return current
}

// mergeConsumer applies only the consumer fields present in an update.
func mergeConsumer(current, update ConsumerState) ConsumerState {
	current.Power = mergeField(current.Power, update.Power)
	current.Energy = mergeField(current.Energy, update.Energy)
	return current
}

// mergeGrid applies present meter fields and detaches the phase-current slice.
func mergeGrid(current, update GridState) GridState {
	current.Power = mergeField(current.Power, update.Power)
	current.Energy = mergeField(current.Energy, update.Energy)
	current.Currents = mergeField(current.Currents, update.Currents)
	if current.Currents.Set && !current.Currents.Null {
		current.Currents.Value = cloneSlice(current.Currents.Value)
	}
	return current
}

// mergeChargePoint applies loadpoint fields and clears vehicle telemetry when its association changes.
func mergeChargePoint(current, update ChargePointState) ChargePointState {
	// Do not attribute telemetry from a previous vehicle to a replacement or disconnected vehicle.
	vehicleChanged := update.VehicleName.Set && !update.VehicleName.Null && update.VehicleName.Value != "" &&
		current.VehicleName.Set && !current.VehicleName.Null && current.VehicleName.Value != update.VehicleName.Value
	if vehicleChanged {
		current.VehicleSOC = Field[float64]{Set: true, Null: true}
		current.VehicleRange = Field[float64]{Set: true, Null: true}
		current.VehicleOdometer = Field[float64]{Set: true, Null: true}
	}
	current.Name = mergeField(current.Name, update.Name)
	current.Charging = mergeField(current.Charging, update.Charging)
	current.Connected = mergeField(current.Connected, update.Connected)
	current.Power = mergeField(current.Power, update.Power)
	current.ChargedEnergy = mergeField(current.ChargedEnergy, update.ChargedEnergy)
	current.VehicleName = mergeField(current.VehicleName, update.VehicleName)
	current.VehicleSOC = mergeField(current.VehicleSOC, update.VehicleSOC)
	current.VehicleRange = mergeField(current.VehicleRange, update.VehicleRange)
	current.VehicleOdometer = mergeField(current.VehicleOdometer, update.VehicleOdometer)
	if (update.VehicleName.Set && (update.VehicleName.Null || update.VehicleName.Value == "")) ||
		(update.Connected.Set && !update.Connected.Null && !update.Connected.Value) {
		current.VehicleName = Field[string]{Set: true, Null: true}
		current.VehicleSOC = Field[float64]{Set: true, Null: true}
		current.VehicleRange = Field[float64]{Set: true, Null: true}
		current.VehicleOdometer = Field[float64]{Set: true, Null: true}
	}
	return current
}
