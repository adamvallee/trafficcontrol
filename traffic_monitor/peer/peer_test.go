package peer

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/apache/trafficcontrol/v8/lib/go-tc"
)

func TestCrStates(t *testing.T) {
	t.Log("Running Peer Tests")

	text, err := ioutil.ReadFile("crstates.json")
	if err != nil {
		t.Log(err)
	}
	crStates, err := tc.CRStatesUnMarshall(text)
	if err != nil {
		t.Log(err)
	}
	fmt.Println(len(crStates.Caches), "caches found")
	for cacheName, crState := range crStates.Caches {
		t.Logf("%v -> %v", cacheName, crState.IsAvailable)
	}

	fmt.Println(len(crStates.DeliveryService), "deliveryservices found")
	for dsName, deliveryService := range crStates.DeliveryService {
		t.Logf("%v -> %v (len:%v)", dsName, deliveryService.IsAvailable, len(deliveryService.DisabledLocations))
	}

}

// TestCRStatesThreadsafeAtomicOperations tests the new atomic delivery service operations
// that fix the race condition between MonitorConfig and HealthResultManager.
func TestCRStatesThreadsafeAtomicOperations(t *testing.T) {
	crStates := NewCRStatesThreadsafe()
	
	// Test SetDeliveryServiceIfNotExists
	dsName1 := tc.DeliveryServiceName("test-ds-1")
	dsName2 := tc.DeliveryServiceName("test-ds-2")
	ds1 := tc.CRStatesDeliveryService{IsAvailable: false, DisabledLocations: []tc.CacheGroupName{}}
	ds2 := tc.CRStatesDeliveryService{IsAvailable: true, DisabledLocations: []tc.CacheGroupName{}}
	
	// Should set the first time
	if !crStates.SetDeliveryServiceIfNotExists(dsName1, ds1) {
		t.Error("Expected SetDeliveryServiceIfNotExists to return true when setting new delivery service")
	}
	
	// Should not set the second time (already exists)
	if crStates.SetDeliveryServiceIfNotExists(dsName1, ds2) {
		t.Error("Expected SetDeliveryServiceIfNotExists to return false when delivery service already exists")
	}
	
	// Verify the original value wasn't overwritten
	if result, exists := crStates.GetDeliveryService(dsName1); !exists || result.IsAvailable != false {
		t.Error("SetDeliveryServiceIfNotExists overwrote existing delivery service")
	}
	
	// Add another delivery service
	crStates.SetDeliveryServiceIfNotExists(dsName2, ds2)
	
	// Test DeleteDeliveryServicesNotIn
	keepSet := map[string]struct{}{
		string(dsName1): {},
		// Note: dsName2 is NOT in the keep set, so it should be deleted
	}
	
	deleted := crStates.DeleteDeliveryServicesNotIn(keepSet)
	
	// Should have deleted dsName2
	if len(deleted) != 1 || deleted[0] != dsName2 {
		t.Errorf("Expected to delete dsName2, but deleted: %v", deleted)
	}
	
	// dsName1 should still exist
	if _, exists := crStates.GetDeliveryService(dsName1); !exists {
		t.Error("DeleteDeliveryServicesNotIn deleted a delivery service that should have been kept")
	}
	
	// dsName2 should be gone
	if _, exists := crStates.GetDeliveryService(dsName2); exists {
		t.Error("DeleteDeliveryServicesNotIn did not delete a delivery service that should have been removed")
	}
}
