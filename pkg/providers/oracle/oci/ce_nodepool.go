package oci

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/containerengine"
)

// CreateNodePool creates node pool specified in the request
func (ce *ContainerEngine) CreateNodePool(request containerengine.CreateNodePoolRequest) (nodepoolOCID string, err error) {

	ctx := context.Background()

	response, err := ce.client.CreateNodePool(ctx, request)
	if err != nil {
		return nodepoolOCID, err
	}

	workReqResp, err := ce.waitUntilWorkRequestComplete(*ce.client, response.OpcWorkRequestId)
	if err != nil {
		return nodepoolOCID, err
	}

	if workReqResp.WorkRequest.Status != containerengine.WorkRequestStatusSucceeded {
		return nodepoolOCID, fmt.Errorf("WorkReqResp status: %s", workReqResp.WorkRequest.Status)
	}

	if workReqResp.WorkRequest.Status == containerengine.WorkRequestStatusSucceeded {
		nodepoolOCID = *ce.getResourceID(workReqResp.Resources, containerengine.WorkRequestResourceActionTypeCreated, "NODEPOOL")
	}

	return nodepoolOCID, err
}

// UpdateNodePool updates a node pool specified in a request
func (ce *ContainerEngine) UpdateNodePool(request containerengine.UpdateNodePoolRequest) (nodepoolOCID string, err error) {

	response, err := ce.client.UpdateNodePool(context.Background(), request)
	if err != nil {
		return nodepoolOCID, err
	}

	workReqResp, err := ce.waitUntilWorkRequestComplete(*ce.client, response.OpcWorkRequestId)
	if err != nil {
		return nodepoolOCID, err
	}

	if workReqResp.WorkRequest.Status != containerengine.WorkRequestStatusSucceeded {
		return nodepoolOCID, fmt.Errorf("WorkReqResp status: %s", workReqResp.WorkRequest.Status)
	}

	if workReqResp.WorkRequest.Status == containerengine.WorkRequestStatusSucceeded {
		nodepoolOCID = *ce.getResourceID(workReqResp.Resources, containerengine.WorkRequestResourceActionTypeUpdated, "NODEPOOL")
	}

	return nodepoolOCID, err
}

// DeleteNodePool deletes a node pool by id
func (ce *ContainerEngine) DeleteNodePool(id *string) error {

	response, err := ce.client.DeleteNodePool(context.Background(), containerengine.DeleteNodePoolRequest{
		NodePoolId: id,
	})
	if err != nil {
		return err
	}

	workReqResp, err := ce.waitUntilWorkRequestComplete(*ce.client, response.OpcWorkRequestId)
	if err != nil {
		return err
	}

	if workReqResp.WorkRequest.Status != containerengine.WorkRequestStatusSucceeded {
		return fmt.Errorf("WorkReqResp status: %s", workReqResp.WorkRequest.Status)
	}

	return nil
}

// DeleteNodePoolByName deletes a node pool in a cluster by name
func (ce *ContainerEngine) DeleteNodePoolByName(clusterID *string, name string) error {

	nodePool, err := ce.GetNodePoolByName(clusterID, name)
	if err != nil {
		return err
	}

	if nodePool.Id == nil {
		return nil
	}

	ce.oci.GetLogger().Infof("Deleting NodePool[%s]", *nodePool.Name)
	ce.DeleteNodePool(nodePool.Id)

	return nil
}

// GetNodePool gets a Node Pool by id
func (ce *ContainerEngine) GetNodePool(id *string) (nodepool containerengine.NodePool, err error) {

	response, err := ce.client.GetNodePool(context.Background(), containerengine.GetNodePoolRequest{
		NodePoolId: id,
	})

	if err != nil {
		return nodepool, err
	}

	return response.NodePool, nil
}

// GetNodePoolByName gets a Node Pool by name within a Cluster
func (ce *ContainerEngine) GetNodePoolByName(clusterID *string, name string) (nodepool containerengine.NodePoolSummary, err error) {

	request := containerengine.ListNodePoolsRequest{
		CompartmentId: common.String(ce.CompartmentOCID),
		ClusterId:     clusterID,
		Name:          common.String(name),
	}

	response, err := ce.client.ListNodePools(context.Background(), request)
	if err != nil {
		return nodepool, err
	}

	if len(response.Items) < 1 {
		return nodepool, &EntityNotFoundError{
			Type: "Node Pool",
			Id:   name,
		}
	}

	return response.Items[0], err
}

// GetNodePools gets all Node Pools within a Cluster
func (ce *ContainerEngine) GetNodePools(clusterID *string) (nodepools []containerengine.NodePoolSummary, err error) {

	request := containerengine.ListNodePoolsRequest{
		CompartmentId: common.String(ce.CompartmentOCID),
		ClusterId:     clusterID,
	}
	request.Limit = common.Int(20)

	listFunc := func(request containerengine.ListNodePoolsRequest) (containerengine.ListNodePoolsResponse, error) {
		return ce.client.ListNodePools(context.Background(), request)
	}

	for r, err := listFunc(request); ; r, err = listFunc(request) {
		if err != nil {
			return nodepools, err
		}

		for _, item := range r.Items {
			nodepools = append(nodepools, item)
		}

		if r.OpcNextPage != nil {
			// if there are more items in next page, fetch items from next page
			request.Page = r.OpcNextPage
		} else {
			// no more result, break the loop
			break
		}
	}

	return nodepools, err
}

// IsNodePoolActive checks whether every node is in ACTIVE not DELETED state in a node pool
func (ce *ContainerEngine) IsNodePoolActive(id *string) bool {

	np, err := ce.GetNodePool(id)
	if err != nil {
		return false
	}

	neededCount := len(np.SubnetIds) * *np.QuantityPerSubnet

	activeNodes := 0
	for _, n := range np.Nodes {
		if n.LifecycleState == containerengine.NodeLifecycleStateDeleted {
			continue
		}
		if n.LifecycleState == containerengine.NodeLifecycleStateActive {
			activeNodes++
		} else {
			ce.oci.logger.Debugf("Node state: %s (%s)", n.LifecycleState, *n.LifecycleDetails)
			break
		}
	}

	if activeNodes == neededCount {
		ce.oci.logger.Infof("All nodes are in ACTIVE state in NodePool[%s]", *np.Name)
		return true
	}

	return false
}

// GetDefaultNodePoolOptions gets default node pool options
func (ce *ContainerEngine) GetDefaultNodePoolOptions() (options NodePoolOptions, err error) {

	return ce.GetNodePoolOptions("all")
}

// GetNodePoolOptions gets available node pool options for a specified cluster OCID
func (ce *ContainerEngine) GetNodePoolOptions(clusterID string) (options NodePoolOptions, err error) {

	request := containerengine.GetNodePoolOptionsRequest{
		NodePoolOptionId: &clusterID,
	}

	r, err := ce.client.GetNodePoolOptions(context.Background(), request)

	return NodePoolOptions{
		Images:             Strings{strings: r.Images},
		KubernetesVersions: Strings{strings: r.KubernetesVersions},
		Shapes:             Strings{strings: r.Shapes},
	}, err
}
