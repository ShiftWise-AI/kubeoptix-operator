# Publicar nova versão do operator

Use este passo a passo simples para buildar e publicar uma nova imagem do operador.

## 1) Ajustar a versão

No projeto, atualize a tag da imagem no manifest do operador, se necessário.

Exemplo:

```bash
cd /home/parraes/redhat/shiftwise-ai/kubeoptix-operator
```

## 2) Build da imagem

```bash
podman build -t quay.io/parraes/shiftwise-operator-system:0.1.3 .
```

Se quiser outra versão:

```bash
podman build -t quay.io/parraes/shiftwise-operator-system:0.1.4 .
```

## 3) Enviar para o registry

```bash
podman push quay.io/parraes/shiftwise-operator-system:0.1.3
```

## 4) Atualizar o manifest de instalação

Se o cluster usar o manifest pronto em `config/manifests/shiftwise-operator.yaml`, verifique se a imagem aponta para a nova tag.

Exemplo:

```yaml
image: quay.io/parraes/shiftwise-operator-system:0.1.3
```

## 5) Instalar/atualizar no cluster

```bash
oc apply -f config/manifests/install-crd.yaml
oc apply -f config/manifests/shiftwise-operator.yaml
```

## 6) Validar

```bash
oc get pods -n shiftwise-ai
oc get deployment -n shiftwise-ai
```

Se o pod estiver Running, a nova versão foi publicada e implantada com sucesso.
