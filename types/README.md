# how to generator deepCopy code

> We used the `k8s.io/code-generator`  generator deepCopy code

1. clone code on `$GOPATH/src/github.com/sealerio/sealer` dir

2. in api root dir add some comments in `doc.go`

   ```go
   // +k8s:deepcopy-gen=package
   // +k8s:defaulter-gen=TypeMeta
   // +groupName=sealer.aliyun.com

   package v1
   ```

   - `+k8s:deepcopy-gen=package` define your generator code is base package
   - `+k8s:defaulter-gen=TypeMeta` default base `TypeMeta`  generator
   - `groupName` is you api groupName , groupVersion is `sealer.aliyun.com/v1`

3. in your type define code need add comments

   ```go
   // +kubebuilder:object:root=true
   // +kubebuilder:subresource:status
   // +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

   // Config is the Schema for the configs API
   type Config struct {
   	metav1.TypeMeta   `json:",inline"`
   	metav1.ObjectMeta `json:"metadata,omitempty"`

   	Spec   ConfigSpec   `json:"spec,omitempty"`
   	Status ConfigStatus `json:"status,omitempty"`
   }

   // +kubebuilder:object:root=true
   // +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

   // ConfigList contains a list of Config
   type ConfigList struct {
   	metav1.TypeMeta `json:",inline"`
   	metav1.ListMeta `json:"metadata,omitempty"`
   	Items           []Config `json:"items"`
   }
   ```

   - in root object add comments `+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object`

4. in sealer code exec `make deepcopy`
