package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// Fact represents observed facts
type Fact struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   FactSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status FactStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// FactSpec is the specification of a Fact
type FactSpec struct {
	Name             string        `json:"name" protobuf:"bytes,1,opt,name=name"`
	ID               int           `json:"id" protobuf:"bytes,2,opt,name=id"`
	FactType         string        `json:"factType" protobuf:"bytes,3,opt,name=factType"`
	Measurements     []Measurement `json:"measurements" protobuf:"bytes,4,opt,name=measurements"`
	Statements       []Statement   `json:"statements" protobuf:"bytes,5,opt,name=statements"`
	Original         Original      `json:"original,omitempty" protobuf:"bytes,6,opt,name=original"`
	Tags             []string      `json:"tags,omitempty" protobuf:"bytes,7,opt,name=tags"`
	SubjectReference string        `json:"subject" protobuf:"bytes,8,opt,name=source"`
}

// FactStatus is the status for an Environment resource
type FactStatus struct {
	Version string `json:"version,omitempty" protobuf:"bytes,1,opt,name=version"`
}

// Measurement contains the value measured on this fact
type Measurement struct {
	Name             string   `json:"name" protobuf:"bytes,1,opt,name=name"`
	MeasurementType  string   `json:"measurementType" protobuf:"bytes,2,opt,name=measurementType"`
	MeasurementValue int      `json:"measurementValue" protobuf:"bytes,3,opt,name=measurementValue"`
	Tags             []string `json:"tags,omitempty" protobuf:"bytes,4,opt,name=tags"`
}

// Statement is the Fact statement
type Statement struct {
	Name             string   `json:"name" protobuf:"bytes,1,opt,name=name"`
	StatementType    string   `json:"statementType" protobuf:"bytes,2,opt,name=statementType"`
	MeasurementValue bool     `json:"measurementValue" protobuf:"bytes,3,opt,name=measurementValue"`
	Tags             []string `json:"tags,omitempty" protobuf:"bytes,4,opt,name=tags"`
}

// Original contains the report
type Original struct {
	MimeType string   `json:"mimetype,omitempty" protobuf:"bytes,1,opt,name=mimetype"`
	URL      string   `json:"url,omitempty" protobuf:"bytes,1,opt,name=url"`
	Tags     []string `json:"tags,omitempty" protobuf:"bytes,8,opt,name=tags"`
}

// Recommended measurements for static program analysis
const (
	StaticProgramAnalysisTotalClasses   = "TotalClasses"
	StaticProgramAnalysisTotalBugs      = "TotalBugs"
	StaticProgramAnalysisHighPriority   = "High"
	StaticProgramAnalysisNormalPriority = "Normal"
	StaticProgramAnalysisLowPriority    = "Low"
	StaticProgramAnalysisIgnored        = "Ignored"
)

// Recommended measurements for code coverage
const (
	CodeCoverageMeasurementTotal    = "Total"
	CodeCoverageMeasurementMissed   = "Missed"
	CodeCoverageMeasurementCoverage = "Coverage"
)

// Recommended types for code coverage count
const (
	CodeCoverageCountTypeInstructions = "Instructions"
	CodeCoverageCountTypeBranches     = "Branches"
	CodeCoverageCountTypeComplexity   = "Complexity"
	CodeCoverageCountTypeLines        = "Lines"
	CodeCoverageCountTypeMethods      = "Methods"
	CodeCoverageCountTypeClasses      = "Classes"
)

const (
	MeasurementPercent = "percent"
	MeasurementCount   = "count"
)

const (
	FactTypeCoverage              = "jx.coverage"
	FactTypeStaticProgramAnalysis = "jx.staticProgramAnalysis"
)
