package utils

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

func GetJobIsFinished(job batchv1.Job) bool {
	_, conditionType := GetJobFinishedStatus(job)
	return conditionType != ""
}

func GetJobFinishedStatus(job batchv1.Job) (result bool, conditionType batchv1.JobConditionType) {
	for _, condition := range job.Status.Conditions {
		correctConditionType := (condition.Type == batchv1.JobComplete || condition.Type == batchv1.JobFailed)

		if correctConditionType && condition.Status == corev1.ConditionTrue {
			result = true
			conditionType = condition.Type
			break
		}
	}

	return result, conditionType
}
