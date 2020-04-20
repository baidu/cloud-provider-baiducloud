package cloud_provider

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"

	"icode.baidu.com/baidu/jpaas-caas/bce-sdk-go/blb"
)

func (bc *Baiducloud) ensureBLB(ctx context.Context, clusterName string, service *v1.Service) (*blb.LoadBalancer, error) {
	startTime := time.Now()
	serviceKey := fmt.Sprintf("%s/%s", service.Namespace, service.Name)
	defer func() {
		klog.V(4).Infof(Message(ctx, fmt.Sprintf("Finished ensureBLB for service %q (%v)", serviceKey, time.Since(startTime))))
	}()
	lb, exist, err := bc.getServiceAssociatedBLB(ctx, clusterName, service)
	if err != nil {
		return nil, err
	}
	if exist && lb != nil {
		msg := fmt.Sprintf("BLB for service %s already exist", serviceKey)
		klog.Info(Message(ctx, msg))
		return lb, nil
	}

	klog.Info(Message(ctx, fmt.Sprintf("BLB for service %s not exist, need create one", serviceKey)))
	lbId, err := bc.createBLB(ctx, service)
	if err != nil || lbId == "" {
		msg := fmt.Sprintf("create BLB for service %s failed: %v", serviceKey, err)
		klog.Error(Message(ctx, msg))
		return nil, fmt.Errorf(msg)
	}

	time.Sleep(6 * time.Second)
	lb, exist, err = bc.getBLBByID(ctx, lbId)
	if err != nil || !exist {
		msg := fmt.Sprintf("check blb %s for service %s exist failed, exist is %v, err is %v", lbId, serviceKey, exist, err)
		klog.Error(Message(ctx, msg))
		return nil, fmt.Errorf(msg)
	}

	if service.Annotations == nil {
		service.Annotations = make(map[string]string, 0)
	}
	service.Annotations[ServiceAnnotationCceAutoAddLoadBalancerID] = lb.BlbId
	service.Annotations[ServiceAnnotationLoadBalancerId] = lb.BlbId
	return lb, nil
}

func (bc *Baiducloud) createBLB(ctx context.Context, service *v1.Service) (string, error) {
	startTime := time.Now()
	serviceKey := fmt.Sprintf("%s/%s", service.Namespace, service.Name)
	defer func() {
		klog.V(4).Infof(Message(ctx, fmt.Sprintf("Finished createBLB for service %q (%v)", serviceKey, time.Since(startTime))))
	}()
	vpcID, subnetID, err := bc.getVpcInfoForBLB(ctx, service)
	if err != nil {
		return "", fmt.Errorf(" Can't get VPC info for BLB: %v\n ", err)
	}

	allocateVip := false
	if allocate, ok := service.Annotations[ServiceAnnotationLoadBalancerAllocateVip]; ok && allocate == "true" {
		allocateVip = true
		klog.Infof(Message(ctx, fmt.Sprintf("allocateVip for service %s", serviceKey)))
	}
	blbName := getBlbName(bc.ClusterID, service)
	args := blb.CreateLoadBalancerArgs{
		Name:        blbName,
		VpcID:       vpcID,
		SubnetID:    subnetID,
		Desc:        "auto generated by cce:" + bc.ClusterID,
		AllocateVIP: allocateVip,
	}
	klog.Infof(Message(ctx, fmt.Sprintf("create blb for service %s args: %v", serviceKey, args)))
	resp, err := bc.clientSet.BLBClient.CreateLoadBalancer(ctx, &args, bc.getSignOption(ctx))
	if err != nil {
		return "", err
	}
	klog.Infof(Message(ctx, fmt.Sprintf("create blb for service %s success, BLB name: %s, BLB id: %s, BLB address: %s.", serviceKey, resp.Name, resp.LoadBalancerId, resp.Address)))
	return resp.LoadBalancerId, nil
}

func (bc *Baiducloud) getServiceAssociatedBLB(ctx context.Context, clusterName string, service *v1.Service) (*blb.LoadBalancer, bool, error) {
	bc.workAround(ctx, clusterName, service)
	result, err := ExtractServiceAnnotation(service)
	if err != nil {
		return nil, false, err
	}

	ID := result.LoadBalancerID
	autoAddID := result.CceAutoAddLoadBalancerID
	existID := result.LoadBalancerExistID
	if ID != "" {
		klog.Infof(Message(ctx, fmt.Sprintf("BLB ID %s is set in annotation", existID)))
		lb, exist, err := bc.getBLBByID(ctx, ID)
		if err != nil {
			return nil, false, err
		}
		if exist {
			klog.Infof(Message(ctx, fmt.Sprintf("getServiceAssociatedBLB by ID %s in annotation, lb is %+v", ID, lb)))
			return lb, exist, nil
		}
	}

	if existID != "" {
		klog.Infof(Message(ctx, fmt.Sprintf("BLB existID %s is set in annotation", existID)))
		lb, exist, err := bc.getBLBByID(ctx, existID)
		if err != nil {
			return nil, false, err
		}
		if exist {
			klog.Infof(Message(ctx, fmt.Sprintf("getServiceAssociatedBLB by existID %s in annotation, lb is %+v", existID, lb)))
			// 兼容 existID annotation 的老集群
			service.Annotations[ServiceAnnotationLoadBalancerReserveLB] = "true"
			return lb, exist, nil
		}
	}

	if autoAddID != "" {
		klog.Infof(Message(ctx, fmt.Sprintf("BLB autoAddID %s is set in annotation", autoAddID)))
		lb, exist, err := bc.getBLBByID(ctx, autoAddID)
		if err != nil {
			return nil, false, err
		}
		if exist {
			klog.Infof(Message(ctx, fmt.Sprintf("getServiceAssociatedBLB by autoAddID %s in annotation, lb is %+v", autoAddID, lb)))
			return lb, exist, nil
		}
	}

	blbName := getBlbName(bc.ClusterID, service)
	klog.Infof(Message(ctx, fmt.Sprintf("try to getServiceAssociatedBLB by blb name %s", blbName)))
	return bc.getBLBByName(ctx, blbName)
}

func (bc *Baiducloud) ensureBLBDeleted(ctx context.Context, lb *blb.LoadBalancer) error {
	if lb == nil {
		klog.Warningf(Message(ctx, fmt.Sprintf("lb is nil, skip delete")))
		return nil
	}
	return bc.clientSet.BLBClient.DeleteLoadBalancer(
		ctx,
		&blb.DeleteLoadBalancerArgs{LoadBalancerId: lb.BlbId},
		bc.getSignOption(ctx))
}
