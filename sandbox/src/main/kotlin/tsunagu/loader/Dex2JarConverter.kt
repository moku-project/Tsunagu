package tsunagu.loader

import com.googlecode.d2j.dex.Dex2jar
import com.googlecode.d2j.reader.MultiDexFileReader
import com.googlecode.dex2jar.tools.BaksmaliBaseDexExceptionHandler
import io.github.oshai.kotlinlogging.KotlinLogging
import java.io.File
import java.nio.file.Files

class Dex2JarConversionException(message: String) : Exception(message)

object Dex2JarConverter {
    private val logger = KotlinLogging.logger {}

    fun convert(apkFile: File): File {
        val outputJar = Files.createTempFile("tsunagu-ext-", ".jar")
        Files.deleteIfExists(outputJar)

        val reader = MultiDexFileReader.open(apkFile.readBytes())
        val handler = BaksmaliBaseDexExceptionHandler()

        Dex2jar
            .from(reader)
            .withExceptionHandler(handler)
            .reUseReg(false)
            .topoLogicalSort()
            .skipDebug(true)
            .optimizeSynchronized(false)
            .printIR(false)
            .noCode(false)
            .skipExceptions(false)
            .dontSanitizeNames(true)
            .computeFrames(true)
            .to(outputJar)

        if (handler.hasException()) {
            val errorFile = Files.createTempFile("tsunagu-ext-error-", ".txt")
            logger.error {
                """
                Detail Error Information in File $errorFile
                Please report this file to one of following link if possible (any one).
                https://sourceforge.net/p/dex2jar/tickets/
                https://bitbucket.org/pxb1988/dex2jar/issues
                https://github.com/pxb1988/dex2jar/issues
                dex2jar@googlegroups.com
                """.trimIndent()
            }
            handler.dump(errorFile, emptyArray<String>())
        } else {
            BytecodeEditor.fixAndroidClasses(outputJar)
        }

        return outputJar.toFile()
    }
}
