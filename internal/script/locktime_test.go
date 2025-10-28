package script

import (
	"testing"
)

func TestOpCheckLocktimeVerify(t *testing.T) {
	tests := []struct {
		name         string
		stackValue   int64
		txLocktime   uint32
		sequence     uint32
		shouldPass   bool
		description  string
	}{
		{
			name:        "block height - pass",
			stackValue:  100,
			txLocktime:  150,
			sequence:    0xfffffffe,
			shouldPass:  true,
			description: "tx locktime (150) >= stack (100), same type (block height)",
		},
		{
			name:        "block height - fail (locktime too early)",
			stackValue:  200,
			txLocktime:  150,
			sequence:    0xfffffffe,
			shouldPass:  false,
			description: "tx locktime (150) < stack (200), should fail",
		},
		{
			name:        "timestamp - pass",
			stackValue:  1600000000,
			txLocktime:  1700000000,
			sequence:    0xfffffffe,
			shouldPass:  true,
			description: "tx locktime (timestamp) >= stack (timestamp)",
		},
		{
			name:        "timestamp - fail (type mismatch)",
			stackValue:  100,
			txLocktime:  1700000000,
			sequence:    0xfffffffe,
			shouldPass:  false,
			description: "stack is block height (100), tx is timestamp - type mismatch",
		},
		{
			name:        "fail - finalized input",
			stackValue:  100,
			txLocktime:  150,
			sequence:    0xffffffff,
			shouldPass:  false,
			description: "sequence is 0xffffffff (finalized), CLTV should fail",
		},
		{
			name:        "fail - negative stack value",
			stackValue:  -1,
			txLocktime:  150,
			sequence:    0xfffffffe,
			shouldPass:  false,
			description: "negative stack value not allowed",
		},
		{
			name:        "equal values - pass",
			stackValue:  100,
			txLocktime:  100,
			sequence:    0xfffffffe,
			shouldPass:  true,
			description: "tx locktime equals stack value, should pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create script: <stackValue> OP_CHECKLOCKTIMEVERIFY
			script := NewScript([]ScriptCommand{
				{Data: EncodeNum(tt.stackValue), IsData: true},
				{Opcode: OP_CHECKLOCKTIMEVERIFY, IsData: false},
			})

			engine := NewScriptEngine(script)
			result := engine.
				WithLocktime(tt.txLocktime).
				WithSequence(tt.sequence).
				Execute([]byte{})

			if result != tt.shouldPass {
				t.Errorf("%s\n  Expected: %v, Got: %v\n  %s",
					tt.name, tt.shouldPass, result, tt.description)
			}
		})
	}
}

func TestOpCheckLocktimeVerifyPreservesStack(t *testing.T) {
	// OP_CHECKLOCKTIMEVERIFY should NOT consume the stack element
	// Script: <100> OP_CHECKLOCKTIMEVERIFY OP_DROP <1>
	// This should succeed - CLTV leaves value on stack for DROP to consume
	// Push 1 at end so final stack has truthy value

	script := NewScript([]ScriptCommand{
		{Data: EncodeNum(100), IsData: true},            // Push 100
		{Opcode: OP_CHECKLOCKTIMEVERIFY, IsData: false}, // Verify (leaves 100 on stack)
		{Opcode: OP_DROP, IsData: false},                // Drop 100
		{Data: EncodeNum(1), IsData: true},              // Push 1 for success
	})

	engine := NewScriptEngine(script)
	result := engine.
		WithLocktime(150).
		WithSequence(0xfffffffe).
		Execute([]byte{})

	if !result {
		t.Error("OP_CHECKLOCKTIMEVERIFY should not consume the stack element")
	}
}

func TestOpCheckSequenceVerify(t *testing.T) {
	// BIP 112 constants
	const (
		SEQUENCE_LOCKTIME_DISABLE_FLAG = uint32(1 << 31)
		SEQUENCE_LOCKTIME_TYPE_FLAG    = uint32(1 << 22)
		SEQUENCE_LOCKTIME_MASK         = 0x0000ffff
	)

	tests := []struct {
		name        string
		stackValue  int64
		sequence    uint32
		shouldPass  bool
		description string
	}{
		{
			name:        "block-based - pass",
			stackValue:  100,
			sequence:    150, // 150 blocks relative lock
			shouldPass:  true,
			description: "sequence (150 blocks) >= stack (100 blocks)",
		},
		{
			name:        "block-based - fail (not aged enough)",
			stackValue:  200,
			sequence:    150,
			shouldPass:  false,
			description: "sequence (150) < stack (200), input hasn't aged enough",
		},
		{
			name:        "time-based - pass",
			stackValue:  int64(100 | SEQUENCE_LOCKTIME_TYPE_FLAG),
			sequence:    150 | SEQUENCE_LOCKTIME_TYPE_FLAG, // 150 * 512 seconds
			shouldPass:  true,
			description: "sequence (150 time units) >= stack (100 time units)",
		},
		{
			name:        "time-based - fail (type mismatch)",
			stackValue:  100, // block-based
			sequence:    150 | SEQUENCE_LOCKTIME_TYPE_FLAG, // time-based
			shouldPass:  false,
			description: "stack is block-based, sequence is time-based - type mismatch",
		},
		{
			name:        "fail - sequence has disable flag",
			stackValue:  100,
			sequence:    150 | SEQUENCE_LOCKTIME_DISABLE_FLAG,
			shouldPass:  false,
			description: "sequence has bit 31 set (BIP 68 disabled), should fail",
		},
		{
			name:        "pass - stack has disable flag",
			stackValue:  int64(100 | SEQUENCE_LOCKTIME_DISABLE_FLAG),
			sequence:    50, // Less than 100, but should succeed
			shouldPass:  true,
			description: "stack has bit 31 set, CSV succeeds immediately",
		},
		{
			name:        "fail - negative stack value",
			stackValue:  -1,
			sequence:    150,
			shouldPass:  false,
			description: "negative stack value not allowed",
		},
		{
			name:        "equal values - pass",
			stackValue:  100,
			sequence:    100,
			shouldPass:  true,
			description: "sequence equals stack value, should pass",
		},
		{
			name:        "max 16-bit value - pass",
			stackValue:  0xffff,
			sequence:    0xffff,
			shouldPass:  true,
			description: "maximum 16-bit value (65535), should pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create script: <stackValue> OP_CHECKSEQUENCEVERIFY
			script := NewScript([]ScriptCommand{
				{Data: EncodeNum(tt.stackValue), IsData: true},
				{Opcode: OP_CHECKSEQUENCEVERIFY, IsData: false},
				{Data: EncodeNum(1), IsData: true}, // Push 1 for success
			})

			engine := NewScriptEngine(script)
			result := engine.
				WithSequence(tt.sequence).
				Execute([]byte{})

			if result != tt.shouldPass {
				t.Errorf("%s\n  Expected: %v, Got: %v\n  %s",
					tt.name, tt.shouldPass, result, tt.description)
			}
		})
	}
}

func TestOpCheckSequenceVerifyPreservesStack(t *testing.T) {
	// OP_CHECKSEQUENCEVERIFY should NOT consume the stack element
	// Script: <100> OP_CHECKSEQUENCEVERIFY OP_DROP <1>

	script := NewScript([]ScriptCommand{
		{Data: EncodeNum(100), IsData: true},             // Push 100
		{Opcode: OP_CHECKSEQUENCEVERIFY, IsData: false}, // Verify (leaves 100 on stack)
		{Opcode: OP_DROP, IsData: false},                 // Drop 100
		{Data: EncodeNum(1), IsData: true},               // Push 1 for success
	})

	engine := NewScriptEngine(script)
	result := engine.
		WithSequence(150).
		Execute([]byte{})

	if !result {
		t.Error("OP_CHECKSEQUENCEVERIFY should not consume the stack element")
	}
}

func TestOpCheckSequenceVerifyPaymentChannel(t *testing.T) {
	// Common use case: payment channel timeout
	// If counterparty doesn't respond, can reclaim funds after N blocks
	//
	// RedeemScript: <100 blocks> OP_CHECKSEQUENCEVERIFY OP_DROP <pubkey> OP_CHECKSIG
	// This ensures the output can't be spent until 100 blocks after it was confirmed

	relativeBlocks := int64(100)

	script := NewScript([]ScriptCommand{
		{Data: EncodeNum(relativeBlocks), IsData: true},
		{Opcode: OP_CHECKSEQUENCEVERIFY, IsData: false},
		{Opcode: OP_DROP, IsData: false},
		{Data: EncodeNum(1), IsData: true}, // Stand-in for signature check
	})

	// Input with sequence >= 100 should succeed
	engine := NewScriptEngine(script)
	result := engine.
		WithSequence(150). // 150 blocks have passed
		Execute([]byte{})

	if !result {
		t.Error("Payment channel timeout should succeed when enough blocks have passed")
	}

	// Input with sequence < 100 should fail
	engine2 := NewScriptEngine(script)
	result2 := engine2.
		WithSequence(50). // Only 50 blocks have passed
		Execute([]byte{})

	if result2 {
		t.Error("Payment channel timeout should fail when not enough blocks have passed")
	}
}

func TestOpCheckLocktimeVerifyInP2SH(t *testing.T) {
	// Common use case: time-locked P2SH output
	// RedeemScript: <locktime> OP_CHECKLOCKTIMEVERIFY OP_DROP <pubkey> OP_CHECKSIG
	//
	// This allows creating an output that can't be spent until a certain time/block

	locktime := int64(500000) // Block height 500,000

	// This would be the redeemScript in a real P2SH output
	script := NewScript([]ScriptCommand{
		{Data: EncodeNum(locktime), IsData: true},
		{Opcode: OP_CHECKLOCKTIMEVERIFY, IsData: false},
		{Opcode: OP_DROP, IsData: false},
		// In real scenario: <signature> <pubkey> would follow
		{Data: EncodeNum(1), IsData: true}, // Push 1 to make script succeed
	})

	// Transaction with locktime >= 500000 should succeed
	engine := NewScriptEngine(script)
	result := engine.
		WithLocktime(600000).
		WithSequence(0xfffffffe).
		Execute([]byte{})

	if !result {
		t.Error("Time-locked script should succeed when tx locktime >= script locktime")
	}

	// Transaction with locktime < 500000 should fail
	engine2 := NewScriptEngine(script)
	result2 := engine2.
		WithLocktime(400000).
		WithSequence(0xfffffffe).
		Execute([]byte{})

	if result2 {
		t.Error("Time-locked script should fail when tx locktime < script locktime")
	}
}
