package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

type KubernetesStruct struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`

	Metadata struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`

	Spec struct {
		Replicas int `yaml:"replicas"`

		Selector struct {
			MatchLabels struct {
				app string `yaml:"app"`
			} `yaml:"matchLabels"`
		} `yaml:"selector"`

		Template struct {
			Metadata struct {
				Labels struct {
					app string `yaml:"app"`
				} `yaml:"labels"`
			} `yaml:"metadata"`
		} `yaml:"template"`

		Spec struct {
			Containers struct {
				Name  string `yaml:"name"`
				Image string `yaml:"image"`

				Resources struct {
					Limits struct {
						Memory string `yaml:"memory"`
						Cpu    string `yaml:"cpu"`
					} `yaml:"limits"`
				} `yaml:"resources"`

				Ports struct {
					ContainerPort int `yaml:"containerPort"`
				} `yaml:"ports"`
			} `yaml:"containers"`
		} `yaml:"spec"`
	}
}

func getpods(name string, clientset *kubernetes.Clientset) int {

	pods, err := clientset.CoreV1().Pods(name).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	return len(pods.Items)
}

func createDeployment(clientset *kubernetes.Clientset) {
	deploymentsClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "demo",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "web",
							Image: "nginx:1.12",
							Ports: []apiv1.ContainerPort{
								{
									Name:          "http",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}
	fmt.Println("Creating deployment...")
	result, err := deploymentsClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created deployment %q.\n", result.GetObjectMeta().GetName())
	prompt()
}

func updateDeployment(name string, setReplicas int32, setContainer string, clientset *kubernetes.Clientset) {
	deploymentsClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)

	fmt.Println("Updating deployment...")

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := deploymentsClient.Get(context.TODO(), name, metav1.GetOptions{})
		if getErr != nil {
			panic(fmt.Errorf("failed to get latest version of Deployment: %v", getErr))
		}

		result.Spec.Replicas = int32Ptr(setReplicas)
		result.Spec.Template.Spec.Containers[0].Image = setContainer
		_, updateErr := deploymentsClient.Update(context.TODO(), result, metav1.UpdateOptions{})
		return updateErr
	})
	if retryErr != nil {
		panic(fmt.Errorf("update failed: %v", retryErr))
	}
	fmt.Println("Updated deployment...")
	prompt()
}

func deleteDeployment(name string, clientset *kubernetes.Clientset) {
	deploymentsClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)

	fmt.Println("Deleting deployment...")
	deletePolicy := metav1.DeletePropagationForeground
	if err := deploymentsClient.Delete(context.TODO(), name, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		panic(err)
	}
	fmt.Println("Deleted deployment.")
	prompt()
}

func applyyaml() {
	file, err := ioutil.ReadFile("deploy.yaml")

	if err != nil {
		panic(err)
	}

	var kubernetesStruct KubernetesStruct

	err = yaml.Unmarshal(file, &kubernetesStruct)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%#v\n", kubernetesStruct.APIVersion)
	fmt.Printf("%#v\n", kubernetesStruct.Kind)
	fmt.Printf("%#v\n", kubernetesStruct.Metadata.Name)
	fmt.Printf("%#v\n", kubernetesStruct.Spec.Replicas)
	fmt.Printf("%#v\n", kubernetesStruct.Spec.Selector.MatchLabels.app)
	fmt.Printf("%#v\n", kubernetesStruct.Spec.Template.Metadata.Labels.app)
}

func main() {
	// var kubeconfig *string
	// if home := homedir.HomeDir(); home != "" {
	// 	kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	// } else {
	// 	kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	// }
	// flag.Parse()

	// config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	// if err != nil {
	// 	panic(err.Error())
	// }

	// clientset, err := kubernetes.NewForConfig(config)
	// if err != nil {
	// 	panic(err.Error())
	// }

	// pods := getpods("default", clientset)
	// fmt.Println(pods)

	applyyaml()
	// createDeployment(clientset)
	// updateDeployment("demo-deployment", 1, "nginx:1.13", clientset)
	// deleteDeployment("demo-deployment", clientset)
}

func int32Ptr(i int32) *int32 { return &i }

func prompt() {
	fmt.Printf("-> Press Return key to continue.")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		break
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	fmt.Println()
}
