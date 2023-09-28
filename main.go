package main

import (
    "archive/zip"
    "encoding/csv"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "os/exec"
    "sort"
    "time"
    "github.com/joho/godotenv"
)

type Repository struct {
    Name        string    `json:"name"`
    Description string    `json:"description"`
    URL         string    `json:"html_url"`
    UpdatedAt   time.Time `json:"updated_at"`
}

func main() {
    err := godotenv.Load()
    if err != nil {
        panic("Impossible de charger le fichier .env")
    }

    githubToken := os.Getenv("GITHUB_ACCESS_TOKEN")

    // Créez le répertoire 'repositories' s'il n'existe pas
    if _, err := os.Stat("./repositories"); os.IsNotExist(err) {
        err := os.Mkdir("./repositories", 0755)
        if err != nil {
            panic("Impossible de créer le répertoire 'repositories'")
        }
    }

    client := &http.Client{}

    req, err := http.NewRequest("GET", "https://api.github.com/users/ascensian/repos", nil)
    if err != nil {
        panic(err)
    }
    req.Header.Add("Authorization", "token "+githubToken)

    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        panic("La requête a échoué avec le code : " + resp.Status)
    }

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        panic(err)
    }

    var repositories []Repository
    if err := json.Unmarshal(body, &repositories); err != nil {
        panic(err)
    }

    // Triez les dépôts par date de dernière modification (plus récent en premier)
    sort.Slice(repositories, func(i, j int) bool {
        return repositories[i].UpdatedAt.After(repositories[j].UpdatedAt)
    })

    // Écrivez ces données dans un fichier CSV
    file, err := os.Create("repositories.csv")
    if err != nil {
        panic(err)
    }
    defer file.Close()

    writer := csv.NewWriter(file)
    defer writer.Flush()

    // Écrivez l'en-tête CSV
    header := []string{"Nom du dépôt", "Description", "URL du dépôt", "Date de dernière mise à jour"}
    if err := writer.Write(header); err != nil {
        panic(err)
    }

    // Écrivez les données des dépôts dans le CSV
    for _, repo := range repositories {
        row := []string{repo.Name, repo.Description, repo.URL, repo.UpdatedAt.String()}
        if err := writer.Write(row); err != nil {
            panic(err)
        }
    }

    fmt.Println("Données écrites dans repositories.csv.")

    // Cloner les repositories
    for _, repo := range repositories {
        cloneCmd := exec.Command("git", "clone", repo.URL)
        cloneCmd.Dir = "./repositories" // Créez un répertoire 'repositories' pour les cloner
        if err := cloneCmd.Run(); err != nil {
            fmt.Println("Erreur lors du clonage de", repo.Name, ":", err)
        } else {
            fmt.Println("Repository cloné avec succès:", repo.Name)
        }

        // Exécutez un git pull pour obtenir les dernières mises à jour
        pullCmd := exec.Command("git", "pull")
        pullCmd.Dir = "./repositories/" + repo.Name // Assurez-vous que le répertoire existe
        if err := pullCmd.Run(); err != nil {
            fmt.Println("Erreur lors du git pull pour", repo.Name, ":", err)
        } else {
            fmt.Println("git pull réussi pour:", repo.Name)
        }
    }

    // Créez une archive ZIP des repositories clonés
    createZipArchive()
}

func createZipArchive() {
    zipFile, err := os.Create("repositories.zip")
    if err != nil {
        fmt.Println("Erreur lors de la création du fichier ZIP :", err)
        return
    }
    defer zipFile.Close()

    zipWriter := zip.NewWriter(zipFile)
    defer zipWriter.Close()

    // Parcourez les répertoires clonés et ajoutez-les à l'archive ZIP
    dirs, err := ioutil.ReadDir("./repositories")
    if err != nil {
        fmt.Println("Erreur lors de la lecture du répertoire 'repositories' :", err)
        return
    }

    for _, dir := range dirs {
        if dir.IsDir() {
            dirName := dir.Name()
            err := addDirToZip(zipWriter, "./repositories/"+dirName, dirName)
            if err != nil {
                fmt.Println("Erreur lors de l'ajout de", dirName, "à l'archive ZIP :", err)
            }
        }
    }

    fmt.Println("Archive ZIP créée avec succès : repositories.zip")
}

func addDirToZip(zipWriter *zip.Writer, dirPath, baseFolder string) error {
    files, err := ioutil.ReadDir(dirPath)
    if err != nil {
        return err
    }

    for _, file := range files {
        if file.IsDir() {
            // Récursivement ajoutez le sous-répertoire à l'archive
            subDir := dirPath + "/" + file.Name()
            err := addDirToZip(zipWriter, subDir, baseFolder+"/"+file.Name())
            if err != nil {
                return err
            }
            continue
        }

        fileBytes, err := ioutil.ReadFile(dirPath + "/" + file.Name())
        if err != nil {
            return err
        }

        // Créez un chemin relatif pour les fichiers dans l'archive
        relPath := baseFolder + "/" + file.Name()

        fileWriter, err := zipWriter.Create(relPath)
        if err != nil {
            return err
        }

        _, err = fileWriter.Write(fileBytes)
        if err != nil {
            return err
        }
    }

    return nil
}

