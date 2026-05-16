# Refatoração do sistema de folders

## Descrição

Este arquivo serve para auxiliar a IA em como é para ser projetado o sistema de folders no projeto clidocs.

## Workflow

Alguns detalhes antes de mostrar o workflow:

1. Quando abro o programa clidocs existem 3 panels, o folder panel, o snippets panel e o preview panel.
2. No folder panel quando inicio o projeto clidocs ele inicia a visualização de todos os folders do folder pai, no caso se eu somente escrever clidocs no terminal ele me deixa escolher se quero ir para o projeto principal (clidocs_snippets/) ou se desejo escolher outro folder para abrir.
3. Vou usar de exemplo o projeto principal chamado clidocs_snippets para os passos, mas a forma de utilização deve ser implementada não importando qual o folder pai aberto.

### Workflow básico dos folders

1. Quando eu abrir o projeto, mostra no folder panel todos os folders dentro do folder pai aberto (clidocs_snippets)
2. Quando possuir um folder sem subfolders (work/) e eu selecionar ele e clicar em Enter, o folder panel vai acessar esse diretório, mostrando ele com "~/" e os snippets dentro dele.
3. Quando possuir um folder com subfolders (work/examples/) e eu selecionar ele e clicar em Enter, deve abrir um modal chamado "Select the subfolder" onde mostra os folders internos do diretório work que eu posso navegar entre eles para escolher qual eu quero acessar e com Enter eu entro no subfolder. Deve me deixar ir navegando entre os subfolders internos também.
4. Quando estou em qualquer folder e estou focado no folder panel e clicar em "d" minusculo no teclado, ele deve me mostrar o modal de criação de folder.
5. Quando estou selecionando um folder em qualquer folder e clico em "D" maiusculo no teclado, ele deve me mostrar o modal de criação de subfolder.


## Exemplo visual do workflow

### Interagindo entre os folders 

#### Sem subfolders

- folder panel no folder pai:

```txt
# Abrindo o folder pai mostra os seguintes projetos de exemplo
# Obs: deve possuir os icones de folders como já está implementado, abaixo é uma lista de folders

Work
Examples > 
Directories > 

# No exemplo acima o simbolo > mostra que tem subfolders
```

- Após acessar o folder Work:

```txt
# Quando selecionado o ~/ nessa visualização,
# vai mostrar todos os arquivos textos no snippet panels.
# Deve sempre estar ativo o primeiro diretório do folder panel 
# para ver os snippets dele no snippet panel.

~/

```

#### Com subfolders

- folder panel no folder pai:

```txt

# Abrindo o folder pai mostra os seguintes projetos de exemplo
# Obs: deve possuir os icones de folders como já implementado, abaixo uma lista de folders

Work
Examples > 
Directories > 

# No exemplo acima o simbolo > mostra que tem subfolders
```

- Clicando em Enter no diretório Examples:

1. Abre um modal mostrando os diretórios internos

```txt
# Isto é somente um exemplo de modal:

Select the subfolder to open

> Examples/Studies/
Examples/Workflows/
Examples/Enable/

Enter Select | -> Access subfolders | <- Back to parent | q Exit

# > é o mesmo que no folder panel, para selecionar
# Se clicar em enter no subfolder selecionado ele abre no folder panel
# Se clicar em -> (seta para direita) ele mostra os subfolders dentro do subfolder
# Se clicar em <- (seta para esquerda) ele volta um subfolder e pode ir até o folder pai que o projeto foi aberto.
# Exemplo após selecionar o -> :

Select the subfolder to open

> Examples/Studies/my-folder1/
Examples/Studies/my-folder2/

Enter Select | -> Access subfolders | <- Back to parent | q Exit

# Obs: posso ir com o -> até o ultimo subfolder existente
# Obs2: posso ir com o <- até o folder pai quando abrir onde abri o programa

```

2. O modal após a seleção do usuário fecha e altera a visualização do folder panel.
3. Folder panel fica com o nome do folder pai desse subfolder.
4. "~/" mostra os snippets do folder pai.
5. Mostra todos os diretórios do folder atual.


### Criando novos folders

1. no folder panel, quando clicar em "n" minusculo ele abre o modal de criar folder.
2. no folder panel, quando clicar em "N" maiusculo ele abre o modal de criar subfolder no atual selecionado.
3. Se eu acessar um subfolder, essa lógica deve continuar funcionando.

### Favoritando folders

1. no folder panel, quando clicar em "d" minusculo ele favorita o folder selecionado.
2. no folder panel, quando clicar em "D" maiusculo ele abre o modal de favoritos.
3. deve ser possivel favoritar qualquer folder ou subfolder.
4. deve mostrar no modal dos favoritos o caminho até o folder favoritado, começando pelo folder pai.

### Modal de folders favoritos

1. Deve mostrar qual favorito desejo selecionar com Enter, como ja é padrão do sistema.
2. Se eu favoritei o folder pai e um subfolder, deve ter os dois caminhos na lista.
3. Deve mostrar todo o caminho até o folder, para caso tenha folders de mesmo nome em diferentes localizações.
4. Clicando em "o" minusculo no modal ele abre o folder pelo sistema do windows.
5. Clicando em "Enter" ele acessa o favorito, onde muda o folder panel para o folder selecionado.
6. No folder panel, mostra "~/" como o folder que foi selecionado com a estrela do lado, para saber que é favorito.

### Modal de troca de localização

1. Quando estivermos no folder panel em qualquer diretório ele deve ter a opção de clicar em "o" minusculo.
2. Quando clicamos em "o" minusculo ele deve abrir um modal mostrando a localização atual do diretório pai aberto (já existe).
3. Nesse modal deve ter a opção de trocar a localização, ou seja, posso abrir meu program em outro folder qualquer do computador.
4. Quando abrir outro folder, ele vai se tornar o folder pai do software.

### Delete de folder

1. Quando clicamos no "x" minusculo do teclado em um folder selecionado, ele deve abrir o modal de deleção.
2. No modal ele mostra o nome do folder e pergunta se queremos deletar, pedindo confirmação.
3. Deve ser possivel deletar qualquer folder em quanquer folder pai ou subfolder, sempre mostrando o caminho até o folder.
4. Se clicarmos "X" maiusculo no folder panel ele deixa selecionar mais de um folder para deletar.
5. Quando estivermos selecionando mais de um, o nome do folder que vai ser deletado deve ficar em vermelho.
6. Usamos a tecla "Space" para ir selecionando os folders.
7. Quando clicarmos em "Enter" vai abrir um modal mostrando todos os folders selecionados para confirmar a deleção.

### Rename de folder

1. Quando clicamos em "r" minusculo do teclado em um folder, vai abrir o model para renomear o folder.
2. Deve existir uma validação na hora de renomear onde somente pode deixar renomear se o nome não for separado (nome subnome/ não pode).
3. Deve mostrar uma mensagem de erro caso tente salvar nome separado, deve ser "user_name" ou "user-name".
4. Quando se tiver esse folder nos favoritos, deve mudar o nome na lista de favoritos também.

### Busca de folder

1. No folder panel quando clicamos em "/" no teclado ele deve abrir um modal chamado "Search Folder".
2. Deve ter uma área para poder escrever, onde enquanto vou digitando ele vai apresentando embaixo todos os folders que tem as letras que vou digitando.
3. Se tenho "Teste" e "Testemunha" e escrevo na busca "te" ele deve mostrar somente os diretórios que começam com essas letras.
4. Deve ser possivel dele buscar dentro de subdiretórios também, onde se tenho "work/teste" ou "work/func/Teste" ele deve mostrar na busca também.
5. Deve ser possivel ativar essa busca clicando em "/" em qualquer folder ou subfolder no software.
6. Quando eu selecionar o folder que desejo e clicar em "Enter" ele deve acessar o folder alterando o folder panel.

### Retorno ao Home

1. Quando trocamos o folder para outro externo, se clicarmos em "H" maiusculo ele voltar para o folder original.



