# Instalar o OpenShift CLI (oc)

Este guia simples instala o `oc` no Linux.

## 1) Baixar o cliente do OpenShift

Acesse a página oficial do OpenShift e baixe a versão correta para o seu sistema, ou use a CLI do cluster se já estiver disponível.

## 2) Extrair e mover para o PATH

Exemplo:

```bash
mkdir -p ~/bin
cd ~/bin
# mova o binário do oc aqui, ou copie do arquivo baixado
chmod +x oc
```

## 3) Verificar instalação

```bash
oc version --client
```

Se aparecer a versão do cliente, a instalação foi bem-sucedida.

## 4) Logar no cluster

```bash
oc login https://api.seu-cluster.example.com:6443
```

Ou com token:

```bash
oc login --token=SEU_TOKEN --server=https://api.seu-cluster.example.com:6443
```

## 5) Verificar acesso

```bash
oc whoami
oc get nodes
```

## 6) Criar namespace de trabalho

```bash
oc new-project shiftwise-ai
```

Se quiser usar um namespace existente:

```bash
oc project shiftwise-ai
```

## Dica

Se o `oc` não estiver no PATH, rode:

```bash
export PATH="$HOME/bin:$PATH"
```

ou adicione isso ao seu `~/.bashrc`.
