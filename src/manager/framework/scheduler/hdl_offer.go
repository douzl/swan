package scheduler

import (
	"github.com/Dataman-Cloud/swan/src/manager/framework/state"
	"github.com/Dataman-Cloud/swan/src/mesosproto/mesos"
	"github.com/Dataman-Cloud/swan/src/mesosproto/sched"

	"github.com/Sirupsen/logrus"
	"github.com/golang/protobuf/proto"
)

func OfferHandler(h *Handler) (*Handler, error) {

	for _, offer := range h.MesosEvent.Event.Offers.Offers {
		logrus.WithFields(logrus.Fields{"handler": "offer"}).Debugf("offerId: %s", *offer.GetId().Value)
		// when no pending offer slot
		if len(h.Manager.SchedulerRef.Allocator.PendingOfferSlots) == 0 {
			RejectOffer(h, offer)

		} else {
			offerWrapper := state.NewOfferWrapper(offer)
			taskInfos := make([]*mesos.TaskInfo, 0)
			for {
				// loop through all pending offer slots
				slot := h.Manager.SchedulerRef.Allocator.PopNextPendingOffer()
				if slot == nil {
					break
				}

				match := slot.TestOfferMatch(offerWrapper)
				if match {
					// TODO the following code logic complex, need improvement
					// offerWrapper cpu/mem/disk deduction recorded within the obj itself
					_, taskInfo := slot.ReserveOfferAndPrepareTaskInfo(offerWrapper)
					h.Manager.SchedulerRef.Allocator.SetOfferSlotMap(offer.GetId(), slot)
					taskInfos = append(taskInfos, taskInfo)
				} else {
					// put the slot back into the queue, in the end
					h.Manager.SchedulerRef.Allocator.PutSlotBackToPendingQueue(slot)
				}
			}

			if len(taskInfos) > 0 {
				LaunchTaskInfos(h, offer, taskInfos)
			}
		}
	}

	return h, nil
}

func LaunchTaskInfos(h *Handler, offer *mesos.Offer, taskInfos []*mesos.TaskInfo) {
	call := &sched.Call{
		FrameworkId: h.Manager.SchedulerRef.MesosConnector.Framework.GetId(),
		Type:        sched.Call_ACCEPT.Enum(),
		Accept: &sched.Call_Accept{
			OfferIds: []*mesos.OfferID{
				offer.GetId(),
			},
			Operations: []*mesos.Offer_Operation{
				&mesos.Offer_Operation{
					Type: mesos.Offer_Operation_LAUNCH.Enum(),
					Launch: &mesos.Offer_Operation_Launch{
						TaskInfos: taskInfos,
					},
				},
			},
			Filters: &mesos.Filters{RefuseSeconds: proto.Float64(1)},
		},
	}

	h.Response.Calls = append(h.Response.Calls, call)
}

func RejectOffer(h *Handler, offer *mesos.Offer) {
	call := &sched.Call{
		FrameworkId: h.Manager.SchedulerRef.MesosConnector.Framework.GetId(),
		Type:        sched.Call_DECLINE.Enum(),
		Decline: &sched.Call_Decline{
			OfferIds: []*mesos.OfferID{
				{
					Value: offer.GetId().Value,
				},
			},
			Filters: &mesos.Filters{
				RefuseSeconds: proto.Float64(1),
			},
		},
	}

	h.Response.Calls = append(h.Response.Calls, call)
}
