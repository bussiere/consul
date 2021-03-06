package consul

import (
	"fmt"
	"github.com/armon/go-metrics"
	"github.com/hashicorp/consul/consul/structs"
)

// Health endpoint is used to query the health information
type Health struct {
	srv *Server
}

// ChecksInState is used to get all the checks in a given state
func (h *Health) ChecksInState(args *structs.ChecksInStateRequest,
	reply *structs.IndexedHealthChecks) error {
	if done, err := h.srv.forward("Health.ChecksInState", args.Datacenter, args, reply); done {
		return err
	}

	// Get the state specific checks
	state := h.srv.fsm.State()
	return h.srv.blockingRPC(&args.BlockingQuery,
		state.QueryTables("ChecksInState"),
		func() (uint64, error) {
			reply.Index, reply.HealthChecks = state.ChecksInState(args.State)
			return reply.Index, nil
		})
}

// NodeChecks is used to get all the checks for a node
func (h *Health) NodeChecks(args *structs.NodeSpecificRequest,
	reply *structs.IndexedHealthChecks) error {
	if done, err := h.srv.forward("Health.NodeChecks", args.Datacenter, args, reply); done {
		return err
	}

	// Get the node checks
	state := h.srv.fsm.State()
	return h.srv.blockingRPC(&args.BlockingQuery,
		state.QueryTables("NodeChecks"),
		func() (uint64, error) {
			reply.Index, reply.HealthChecks = state.NodeChecks(args.Node)
			return reply.Index, nil
		})
}

// ServiceChecks is used to get all the checks for a service
func (h *Health) ServiceChecks(args *structs.ServiceSpecificRequest,
	reply *structs.IndexedHealthChecks) error {
	// Reject if tag filtering is on
	if args.TagFilter {
		return fmt.Errorf("Tag filtering is not supported")
	}

	// Potentially forward
	if done, err := h.srv.forward("Health.ServiceChecks", args.Datacenter, args, reply); done {
		return err
	}

	// Get the service checks
	state := h.srv.fsm.State()
	return h.srv.blockingRPC(&args.BlockingQuery,
		state.QueryTables("ServiceChecks"),
		func() (uint64, error) {
			reply.Index, reply.HealthChecks = state.ServiceChecks(args.ServiceName)
			return reply.Index, nil
		})
}

// ServiceNodes returns all the nodes registered as part of a service including health info
func (h *Health) ServiceNodes(args *structs.ServiceSpecificRequest, reply *structs.IndexedCheckServiceNodes) error {
	if done, err := h.srv.forward("Health.ServiceNodes", args.Datacenter, args, reply); done {
		return err
	}

	// Verify the arguments
	if args.ServiceName == "" {
		return fmt.Errorf("Must provide service name")
	}

	// Get the nodes
	state := h.srv.fsm.State()
	err := h.srv.blockingRPC(&args.BlockingQuery,
		state.QueryTables("CheckServiceNodes"),
		func() (uint64, error) {
			if args.TagFilter {
				reply.Index, reply.Nodes = state.CheckServiceTagNodes(args.ServiceName, args.ServiceTag)
			} else {
				reply.Index, reply.Nodes = state.CheckServiceNodes(args.ServiceName)
			}
			return reply.Index, nil
		})

	// Provide some metrics
	if err == nil {
		metrics.IncrCounter([]string{"consul", "health", "service", "query", args.ServiceName}, 1)
		if args.ServiceTag != "" {
			metrics.IncrCounter([]string{"consul", "health", "service", "query-tag", args.ServiceName, args.ServiceTag}, 1)
		}
		if len(reply.Nodes) == 0 {
			metrics.IncrCounter([]string{"consul", "health", "service", "not-found", args.ServiceName}, 1)
		}
	}
	return err
}
