package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/skhalash/kube-secret-madness/pkg/rand"

	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/workqueue"
)

var workers = 200
var secretsTotal = 100

type flags struct {
	kubeconfig *string
}

func main() {
	var f = flags{}
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig != "" {
		f.kubeconfig = &kubeconfig
	} else if home := homedir.HomeDir(); home != "" {
		f.kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		f.kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	if err := run(&f); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(2)
	}
}

func run(f *flags) error {
	config, err := clientcmd.BuildConfigFromFlags("", *f.kubeconfig)
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	client := clientset.CoreV1().Secrets("secrets")

	ctx := context.Background()

	secrets, err := createSecrets(ctx, client, secretsTotal)
	if err != nil {
		return err
	}

	return updateSecrets(ctx, client, secrets)
}

func createSecrets(ctx context.Context, client v1.SecretInterface, n int) (*corev1.SecretList, error) {
	fmt.Printf("Creating %d secrets...\n", n)
	workqueue.ParallelizeUntil(ctx, workers, n, func(i int) {
		_, err := createSecret(ctx, client)
		if err != nil {
			panic(err)
		}
	})

	secrets, err := client.List(ctx, metav1.ListOptions{LabelSelector: "generated=true"})
	if err != nil {
		return nil, err
	}

	fmt.Printf("There are %d secrets\n", len(secrets.Items))

	return secrets, nil
}

func createSecret(ctx context.Context, client v1.SecretInterface) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: rand.String(5),
			Labels: map[string]string{
				"generated": "true",
			},
		},
		Data: rand.SecretData(),
	}
	return client.Create(ctx, secret, metav1.CreateOptions{})
}

func updateSecrets(ctx context.Context, client v1.SecretInterface, secrets *corev1.SecretList) error {

	g, ctx := errgroup.WithContext(ctx)
	total := len(secrets.Items)
	for i := 0; i < total; i += workers {
		from, to := i, min(i+workers, total)
		fmt.Printf("Updating secrets in the index range from %d to %d\n", from, to)

		g.Go(func() error {
			return updateSecretsInRange(ctx, client, secrets, from, to)
		})
	}

	return g.Wait()
}

func updateSecretsInRange(ctx context.Context, client v1.SecretInterface, secrets *corev1.SecretList, from, to int) error {
	for {
		index := rand.Index(to-from) + from
		oldSecret := secrets.Items[index]
		oldSecret.Data = rand.SecretData()
		newSecret, err := client.Update(ctx, &oldSecret, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		secrets.Items[index] = *newSecret
	}
}

func min(x, y int) int {
	return int(math.Min(float64(x), float64(y)))
}
