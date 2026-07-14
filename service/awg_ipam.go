package service

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net/netip"

	"github.com/deposist/s-ui-x-extended/database/model"

	"gorm.io/gorm"
)

var ErrAWGAddressPoolExhausted = errors.New("AWG address pool is exhausted")

// AllocateAWGIPv4 returns the lowest free client address. The caller must pass
// a write transaction and insert the AWGDevice using the returned address
// before committing that transaction.
func AllocateAWGIPv4(tx *gorm.DB, subnet netip.Prefix, serverAddress netip.Addr, now int64) (netip.Addr, error) {
	if tx == nil {
		return netip.Addr{}, errors.New("AWG IP allocation requires a database transaction")
	}
	if !subnet.IsValid() || !subnet.Addr().Is4() || !serverAddress.Is4() {
		return netip.Addr{}, errors.New("AWG IP allocation requires an IPv4 subnet and server address")
	}
	subnet = subnet.Masked()
	if !subnet.Contains(serverAddress) {
		return netip.Addr{}, errors.New("AWG server address is outside the allocation subnet")
	}

	var rows []struct {
		IPv4Address string
	}
	if err := tx.Model(&model.AWGDevice{}).
		Select("ipv4_address").
		Where("desired_enabled = ? OR provisioned = ? OR sync_state = ? OR ip_reusable_after > ?", true, true, "pending_remove", now).
		Find(&rows).Error; err != nil {
		return netip.Addr{}, err
	}
	used := make(map[netip.Addr]struct{}, len(rows)+2)
	used[serverAddress] = struct{}{}
	for _, row := range rows {
		address, err := netip.ParseAddr(row.IPv4Address)
		if err != nil || !address.Is4() {
			return netip.Addr{}, fmt.Errorf("invalid persisted AWG IPv4 address")
		}
		used[address] = struct{}{}
	}

	network := ipv4Uint32(subnet.Addr())
	hostBits := 32 - subnet.Bits()
	if hostBits < 2 {
		return netip.Addr{}, ErrAWGAddressPoolExhausted
	}
	mask := ^uint32(0) >> (32 - hostBits)
	broadcast := network | mask
	for candidate := network + 1; candidate < broadcast; candidate++ {
		address := uint32IPv4(candidate)
		if _, exists := used[address]; !exists {
			return address, nil
		}
	}
	return netip.Addr{}, ErrAWGAddressPoolExhausted
}

func ipv4Uint32(address netip.Addr) uint32 {
	bytes := address.As4()
	return binary.BigEndian.Uint32(bytes[:])
}

func uint32IPv4(value uint32) netip.Addr {
	var bytes [4]byte
	binary.BigEndian.PutUint32(bytes[:], value)
	return netip.AddrFrom4(bytes)
}
