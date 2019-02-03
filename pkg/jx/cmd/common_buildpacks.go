package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jenkins-x/draft-repo/pkg/draft/pack"
	"github.com/jenkins-x/jx/pkg/config"
	jxdraft "github.com/jenkins-x/jx/pkg/draft"
	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/jenkins-x/jx/pkg/jenkinsfile/gitresolver"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
)

// InvokeDraftPack used to pass arguments into the draft pack invocation
type InvokeDraftPack struct {
	Dir                     string
	CustomDraftPack         string
	Jenkinsfile             string
	DefaultJenkinsfile      string
	WithRename              bool
	InitialisedGit          bool
	DisableJenkinsfileCheck bool
	DisableAddFiles         bool
	ProjectConfig           *config.ProjectConfig
}

// initBuildPacks initalise the build packs
func (o *CommonOptions) initBuildPacks() (string, error) {
	settings, err := o.TeamSettings()
	if err != nil {
		return "", err
	}
	return gitresolver.InitBuildPack(o.Git(), settings.BuildPackURL, settings.BuildPackRef)
}

// invokeDraftPack invokes a draft pack copying in a Jenkinsfile if required
func (o *CommonOptions) invokeDraftPack(i *InvokeDraftPack) (string, error) {
	packsDir, err := o.initBuildPacks()
	if err != nil {
		return "", err
	}

	dir := i.Dir
	customDraftPack := i.CustomDraftPack
	disableJenkinsfileCheck := i.DisableJenkinsfileCheck
	initialisedGit := i.InitialisedGit
	withRename := i.WithRename
	jenkinsfilePath := i.Jenkinsfile
	defaultJenkinsfile := i.DefaultJenkinsfile
	if defaultJenkinsfile == "" {
		defaultJenkinsfile = filepath.Join(dir, jenkins.DefaultJenkinsfile)
	}

	pomName := filepath.Join(dir, "pom.xml")
	gradleName := filepath.Join(dir, "build.gradle")
	jenkinsPluginsName := filepath.Join(dir, "plugins.txt")
	packagerConfigName := filepath.Join(dir, "packager-config.yml")
	envChart := filepath.Join(dir, "env/Chart.yaml")
	lpack := ""
	if len(customDraftPack) == 0 {
		if i.ProjectConfig == nil {
			i.ProjectConfig, _, err = config.LoadProjectConfig(dir)
			if err != nil {
				return "", err
			}
		}
		customDraftPack = i.ProjectConfig.BuildPack
	}

	if len(customDraftPack) > 0 {
		log.Info("trying to use draft pack: " + customDraftPack + "\n")
		lpack = filepath.Join(packsDir, customDraftPack)
		f, err := util.FileExists(lpack)
		if err != nil {
			log.Error(err.Error())
			return "", err
		}
		if f == false {
			log.Error("Could not find pack: " + customDraftPack + " going to try detect which pack to use")
			lpack = ""
		}
	}

	if len(lpack) == 0 {
		if exists, err := util.FileExists(pomName); err == nil && exists {
			pack, err := util.PomFlavour(pomName)
			if err != nil {
				return "", err
			}
			lpack = filepath.Join(packsDir, pack)

			exists, _ = util.FileExists(lpack)
			if !exists {
				log.Warn("defaulting to maven pack")
				lpack = filepath.Join(packsDir, "maven")
			}
		} else if exists, err := util.FileExists(gradleName); err == nil && exists {
			lpack = filepath.Join(packsDir, "gradle")
		} else if exists, err := util.FileExists(jenkinsPluginsName); err == nil && exists {
			lpack = filepath.Join(packsDir, "jenkins")
		} else if exists, err := util.FileExists(packagerConfigName); err == nil && exists {
			lpack = filepath.Join(packsDir, "cwp")
		} else if exists, err := util.FileExists(envChart); err == nil && exists {
			lpack = filepath.Join(packsDir, "environment")
		} else {
			// pack detection time
			lpack, err = jxdraft.DoPackDetectionForBuildPack(o.Out, dir, packsDir)

			if err != nil {
				return "", err
			}
		}
	}
	log.Success("selected pack: " + lpack + "\n")
	draftPack := filepath.Base(lpack)
	i.CustomDraftPack = draftPack

	if i.DisableAddFiles {
		return draftPack, nil
	}

	chartsDir := filepath.Join(dir, "charts")
	jenkinsfileExists, err := util.FileExists(jenkinsfilePath)
	exists, err := util.FileExists(chartsDir)
	if exists && err == nil {
		exists, err = util.FileExists(filepath.Join(dir, "Dockerfile"))
		if exists && err == nil {
			if jenkinsfileExists || disableJenkinsfileCheck {
				log.Warn("existing Dockerfile, Jenkinsfile and charts folder found so skipping 'draft create' step\n")
				return draftPack, nil
			}
		}
	}

	generateJenkinsPath := jenkinsfilePath
	jenkinsfileBackup := ""
	defaultJenkinsfileExists, err := util.FileExists(defaultJenkinsfile)
	if defaultJenkinsfileExists && !disableJenkinsfileCheck {
		// lets copy the old Jenkinsfile in case we override it
		jenkinsfileBackup = defaultJenkinsfile + JenkinsfileBackupSuffix
		err = util.RenameFile(defaultJenkinsfile, jenkinsfileBackup)
		if err != nil {
			return "", fmt.Errorf("Failed to rename old Jenkinsfile: %s", err)
		}
		generateJenkinsPath = defaultJenkinsfile
	}

	err = CopyBuildPack(dir, lpack)
	if err != nil {
		log.Warnf("Failed to apply the build pack in %s due to %s", dir, err)
	}

	if !jenkinsfileExists || jenkinsfileBackup != "" {
		// lets check if we have a pipeline.yaml in the build pack so we can generate one dynamically
		pipelineFile := filepath.Join(lpack, jenkinsfile.PipelineConfigFileName)
		exists, err := util.FileExists(pipelineFile)
		if err != nil {
			return draftPack, err
		}
		if exists {
			modules, err := gitresolver.LoadModules(packsDir)
			if err != nil {
				return draftPack, err
			}

			// lets find the Jenkinsfile template
			tmplFileName := jenkinsfile.PipelineTemplateFileName
			templateFileNames := []string{filepath.Join(lpack, tmplFileName), filepath.Join(packsDir, tmplFileName)}

			moduleResolver, err := gitresolver.ResolveModules(modules, o.Git())
			if err != nil {
				return draftPack, err
			}
			for _, mr := range moduleResolver.Modules {
				templateFileNames = append(templateFileNames, filepath.Join(mr.PacksDir, draftPack, tmplFileName), filepath.Join(mr.PacksDir, tmplFileName))
			}
			templateFile, err := util.FirstFileExists(templateFileNames...)
			if err != nil {
				return draftPack, err
			}
			prow, err := o.isProw()
			if err != nil {
				return draftPack, err
			}

			if templateFile != "" {
				arguments := &jenkinsfile.CreateJenkinsfileArguments{
					ConfigFile:          pipelineFile,
					TemplateFile:        templateFile,
					OutputFile:          generateJenkinsPath,
					JenkinsfileRunner:   prow,
					ClearContainerNames: prow,
				}
				err = arguments.GenerateJenkinsfile(moduleResolver.AsImportResolver())
				if err != nil {
					return draftPack, err
				}
			}
		}
	}

	unpackedDefaultJenkinsfile := defaultJenkinsfile
	if unpackedDefaultJenkinsfile != jenkinsfilePath {
		unpackedDefaultJenkinsfileExists := false
		unpackedDefaultJenkinsfileExists, err = util.FileExists(unpackedDefaultJenkinsfile)
		if unpackedDefaultJenkinsfileExists {
			err = util.RenameFile(unpackedDefaultJenkinsfile, jenkinsfilePath)
			if err != nil {
				return "", fmt.Errorf("Failed to rename Jenkinsfile file from '%s' to '%s': %s", unpackedDefaultJenkinsfile, jenkinsfilePath, err)
			}
			if jenkinsfileBackup != "" {
				err = util.RenameFile(jenkinsfileBackup, defaultJenkinsfile)
				if err != nil {
					return "", fmt.Errorf("Failed to rename Jenkinsfile backup file: %s", err)
				}
			}
		}
	} else if jenkinsfileBackup != "" {
		// if there's no Jenkinsfile created then rename it back again!
		jenkinsfileExists, err = util.FileExists(jenkinsfilePath)
		if err != nil {
			log.Warnf("Failed to check for Jenkinsfile %s", err)
		} else {
			if jenkinsfileExists {
				if !initialisedGit && !withRename{
					err = os.Remove(jenkinsfileBackup)
					if err != nil {
						log.Warnf("Failed to remove Jenkinsfile backup %s", err)
					}
				}
			} else {
				// lets put the old one back again
				err = util.RenameFile(jenkinsfileBackup, jenkinsfilePath)
				if err != nil {
					return "", fmt.Errorf("Failed to rename Jenkinsfile backup file: %s", err)
				}
			}
		}
	}

	return draftPack, nil
}

// CopyBuildPack copies the build pack from the source dir to the destination dir
func CopyBuildPack(dest, src string) error {
	// first do some validation that we are copying from a valid pack directory
	p, err := pack.FromDir(src)
	if err != nil {
		return fmt.Errorf("could not load %s: %s", src, err)
	}

	// lets remove any files we think should be zapped
	for _, file := range []string{jenkinsfile.PipelineConfigFileName, jenkinsfile.PipelineTemplateFileName} {
		delete(p.Files, file)
	}
	return p.SaveDir(dest)
}
