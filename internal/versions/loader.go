package versions

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// zksyncDeploymentType is the deployment type used by ZkSync Era chains.
// It requires a different bytecode format (EraVM) and is not compatible
// with standard EVM chains; therefore unsupported by safekit.
const zksyncDeploymentType = "zksync"

// deploymentJSON mirrors the structure of Safe's deployment JSON files from
// github.com/safe-global/safe-deployments.
type deploymentJSON struct {
	Deployments      map[string]deploymentEntry `json:"deployments"`
	NetworkAddresses networkAddressMap          `json:"networkAddresses"`
	ABI              json.RawMessage            `json:"abi"`
}

// deploymentEntry holds the address and code hash for a single deployment type
// e.g. "canonical" or "eip155".
type deploymentEntry struct {
	Address  string `json:"address"`
	CodeHash string `json:"codeHash"`
}

// networkAddressMap is a map of chain IDs to their preferred deployment types.
//
// Safe's JSON is inconsistent i.e. a chain can map to either a single string
// e.g. "canonical" or an array e.g. ["canonical", "eip155"]. We normalise
// both forms into []string during unmarshalling.
type networkAddressMap map[string][]string

func (m *networkAddressMap) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to parse networkAddresses: %w", err)
	}

	result := make(networkAddressMap, len(raw))
	for chainID, val := range raw {
		var arr []string
		if err := json.Unmarshal(val, &arr); err == nil {
			result[chainID] = arr
			continue
		}

		var str string
		if err := json.Unmarshal(val, &str); err != nil {
			return fmt.Errorf("unexpected format for networkAddresses[%s]", chainID)
		}
		result[chainID] = []string{str}
	}

	*m = result
	return nil
}

// Load parses a Safe deployment JSON file for a specific chain ID and returns
// a ParsedDeployment with the resolved contract address and parsed ABI.
//
// Address resolution follows Safe's preference ordering i.e. when a chain has
// multiple deployment types e.g. ["canonical", "eip155"], we always take
// the first one as Safe intends it to be the preferred type for that chain.
//
// Returns an error if the chain is not present in the JSON, which means Safe
// has not deployed this contract version on that chain.
func Load(jsonBytes []byte, chainID *big.Int) (*ParsedDeployment, error) {
	var d deploymentJSON
	if err := json.Unmarshal(jsonBytes, &d); err != nil {
		return nil, fmt.Errorf("failed to parse deployment JSON: %w", err)
	}

	deploymentTypes, ok := d.NetworkAddresses[chainID.String()]
	if !ok {
		return nil, fmt.Errorf(
			"%w: chain ID %s is not supported in this Safe version", ErrVersionNotOnChain, chainID,
		)
	}
	if len(deploymentTypes) == 0 {
		return nil, fmt.Errorf("%w: chain ID %s has empty deployment types", ErrVersionNotOnChain, chainID)
	}

	preferredType := ""
	for _, dt := range deploymentTypes {
		if !strings.EqualFold(dt, zksyncDeploymentType) {
			preferredType = dt
			break
		}
	}

	if preferredType == "" {
		return nil, fmt.Errorf("%w: chain ID %s has no supported EVM deployment types %v",
			ErrVersionNotOnChain, chainID, deploymentTypes)
	}

	entry, ok := d.Deployments[preferredType]
	if !ok {
		return nil, fmt.Errorf("deployment type %q not found for chain ID %s", preferredType, chainID)
	}
	if entry.Address == "" {
		return nil, fmt.Errorf("empty address for deployment type %q on chain ID %s", preferredType, chainID)
	}

	parsedABI, err := abi.JSON(strings.NewReader(string(d.ABI)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI for chain ID %s: %w", chainID, err)
	}

	return &ParsedDeployment{
		Address: common.HexToAddress(entry.Address),
		ABI:     parsedABI,
	}, nil
}
