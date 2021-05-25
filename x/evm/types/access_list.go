package types

import (
	"github.com/ethereum/go-ethereum/common"
	ethcmn "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// AccessListMappings is copied from go-ethereum
// https://github.com/ethereum/go-ethereum/blob/cf856ea1ad96ac39ea477087822479b63417036a/core/state/access_list.go#L23
type AccessListMappings struct {
	addresses map[common.Address]int
	slots     []map[common.Hash]struct{}
}

// ContainsAddress returns true if the address is in the access list.
func (al *AccessListMappings) ContainsAddress(address common.Address) bool {
	_, ok := al.addresses[address]
	return ok
}

// Contains checks if a slot within an account is present in the access list, returning
// separate flags for the presence of the account and the slot respectively.
func (al *AccessListMappings) Contains(address common.Address, slot common.Hash) (addressPresent bool, slotPresent bool) {
	idx, ok := al.addresses[address]
	if !ok {
		// no such address (and hence zero slots)
		return false, false
	}
	if idx == -1 {
		// address yes, but no slots
		return true, false
	}

	if idx >= len(al.slots) {
		// return in case of out-of-range
		return true, false
	}

	_, slotPresent = al.slots[idx][slot]
	return true, slotPresent
}

// newAccessList creates a new AccessListMappings.
func NewAccessListMappings() *AccessListMappings {
	return &AccessListMappings{
		addresses: make(map[common.Address]int),
	}
}

// Copy creates an independent copy of an AccessListMappings.
func (al *AccessListMappings) Copy() *AccessListMappings {
	cp := NewAccessListMappings()
	for k, v := range al.addresses {
		cp.addresses[k] = v
	}
	cp.slots = make([]map[common.Hash]struct{}, len(al.slots))
	for i, slotMap := range al.slots {
		newSlotmap := make(map[common.Hash]struct{}, len(slotMap))
		for k := range slotMap {
			newSlotmap[k] = struct{}{}
		}
		cp.slots[i] = newSlotmap
	}
	return cp
}

// AddAddress adds an address to the access list, and returns 'true' if the operation
// caused a change (addr was not previously in the list).
func (al *AccessListMappings) AddAddress(address common.Address) bool {
	if _, present := al.addresses[address]; present {
		return false
	}
	al.addresses[address] = -1
	return true
}

// AddSlot adds the specified (addr, slot) combo to the access list.
// Return values are:
// - address added
// - slot added
// For any 'true' value returned, a corresponding journal entry must be made.
func (al *AccessListMappings) AddSlot(address common.Address, slot common.Hash) (addrChange bool, slotChange bool) {
	idx, addrPresent := al.addresses[address]
	if !addrPresent || idx == -1 {
		// Address not present, or addr present but no slots there
		al.addresses[address] = len(al.slots)
		slotmap := map[common.Hash]struct{}{slot: {}}
		al.slots = append(al.slots, slotmap)
		return !addrPresent, true
	}

	if idx >= len(al.slots) {
		// return in case of out-of-range
		return false, false
	}

	// There is already an (address,slot) mapping
	slotmap := al.slots[idx]
	if _, ok := slotmap[slot]; !ok {
		slotmap[slot] = struct{}{}
		// journal add slot change
		return false, true
	}
	// No changes required
	return false, false
}

// DeleteSlot removes an (address, slot)-tuple from the access list.
// This operation needs to be performed in the same order as the addition happened.
// This method is meant to be used  by the journal, which maintains ordering of
// operations.
func (al *AccessListMappings) DeleteSlot(address common.Address, slot common.Hash) {
	idx, addrOk := al.addresses[address]
	// There are two ways this can fail
	if !addrOk {
		panic("reverting slot change, address not present in list")
	}
	slotmap := al.slots[idx]
	delete(slotmap, slot)
	// If that was the last (first) slot, remove it
	// Since additions and rollbacks are always performed in order,
	// we can delete the item without worrying about screwing up later indices
	if len(slotmap) == 0 {
		al.slots = al.slots[:idx]
		al.addresses[address] = -1
	}
}

// DeleteAddress removes an address from the access list. This operation
// needs to be performed in the same order as the addition happened.
// This method is meant to be used  by the journal, which maintains ordering of
// operations.
func (al *AccessListMappings) DeleteAddress(address common.Address) {
	delete(al.addresses, address)
}

// AccessList is an EIP-2930 access list that represents the slice of
// the protobuf AccessTuples.
type AccessList []AccessTuple

// NewAccessList creates a new protobuf-compatible AccessList from an ethereum
// core AccessList type
func NewAccessList(ethAccessList *ethtypes.AccessList) AccessList {
	if ethAccessList == nil {
		return nil
	}

	var AccessListMappings AccessList
	for _, tuple := range *ethAccessList {
		storageKeys := make([]string, len(tuple.StorageKeys))

		for i := range tuple.StorageKeys {
			storageKeys[i] = tuple.StorageKeys[i].String()
		}

		AccessListMappings = append(AccessListMappings, AccessTuple{
			Address:     tuple.Address.String(),
			StorageKeys: storageKeys,
		})
	}

	return AccessListMappings
}

// ToEthAccessList is an utility function to convert the protobuf compatible
// AccessList to eth core AccessList from go-ethereum
func (al AccessList) ToEthAccessList() *ethtypes.AccessList {
	var AccessListMappings ethtypes.AccessList

	for _, tuple := range al {
		storageKeys := make([]ethcmn.Hash, len(tuple.StorageKeys))

		for i := range tuple.StorageKeys {
			storageKeys[i] = ethcmn.HexToHash(tuple.StorageKeys[i])
		}

		AccessListMappings = append(AccessListMappings, ethtypes.AccessTuple{
			Address:     ethcmn.HexToAddress(tuple.Address),
			StorageKeys: storageKeys,
		})
	}

	return &AccessListMappings
}
